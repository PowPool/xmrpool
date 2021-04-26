package stratum

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/MiningPool0826/xmrpool/storage"
	"io"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MiningPool0826/xmrpool/pool"
	"github.com/MiningPool0826/xmrpool/rpc"
	"github.com/MiningPool0826/xmrpool/util"
	. "github.com/MiningPool0826/xmrpool/util"
)

type StratumServer struct {
	luckWindow          int64
	luckLargeWindow     int64
	roundShares         int64
	blockStats          map[int64]blockEntry
	config              *pool.Config
	miners              MinersMap
	blockTemplate       atomic.Value
	upstream            int32
	upstreams           []*rpc.RPCClient
	timeout             time.Duration
	estimationWindow    time.Duration
	hashrateWindow      time.Duration
	hashrateLargeWindow time.Duration

	blocksMu   sync.RWMutex
	sessionsMu sync.RWMutex
	sessions   map[*Session]struct{}

	backend            *storage.RedisClient
	hashrateExpiration time.Duration

	upstreamsStates []bool

	maxConcurrency int
}

type blockEntry struct {
	height   int64
	variance float64
	hash     string
}

type Endpoint struct {
	jobSequence uint64
	config      *pool.Port
	difficulty  *big.Int
	instanceId  []byte
	extraNonce  uint32
	targetHex   string
}

type Session struct {
	lastBlockHeight int64
	sync.Mutex
	conn *net.TCPConn
	enc  *json.Encoder
	ip   string

	login string
	id    string

	endpoint  *Endpoint
	validJobs []*Job
}

const (
	MaxReqSize = 10 * 1024
)

func NewStratum(cfg *pool.Config, backend *storage.RedisClient) *StratumServer {
	stratum := &StratumServer{config: cfg, backend: backend, blockStats: make(map[int64]blockEntry),
		maxConcurrency: cfg.Threads}

	stratum.upstreams = make([]*rpc.RPCClient, len(cfg.Upstream))
	for i, v := range cfg.Upstream {
		client, err := rpc.NewRPCClient(&v)
		if err != nil {
			Error.Fatal(err)
		} else {
			stratum.upstreams[i] = client
			Info.Printf("Upstream: %s => %s", client.Name, client.Url)
		}
	}
	Info.Printf("Default upstream: %s => %s", stratum.rpc().Name, stratum.rpc().Url)

	stratum.miners = NewMinersMap()
	stratum.sessions = make(map[*Session]struct{})

	timeout, _ := time.ParseDuration(cfg.Stratum.Timeout)
	stratum.timeout = timeout

	estimationWindow, _ := time.ParseDuration(cfg.EstimationWindow)
	stratum.estimationWindow = estimationWindow

	hashrateExpiration, _ := time.ParseDuration(cfg.HashRateExpiration)
	stratum.hashrateExpiration = hashrateExpiration

	hashrateWindow, _ := time.ParseDuration(cfg.HashrateWindow)
	stratum.hashrateWindow = hashrateWindow

	hashrateLargeWindow, _ := time.ParseDuration(cfg.HashrateLargeWindow)
	stratum.hashrateLargeWindow = hashrateLargeWindow

	luckWindow, _ := time.ParseDuration(cfg.LuckWindow)
	stratum.luckWindow = int64(luckWindow / time.Millisecond)
	luckLargeWindow, _ := time.ParseDuration(cfg.LargeLuckWindow)
	stratum.luckLargeWindow = int64(luckLargeWindow / time.Millisecond)

	refreshIntv, _ := time.ParseDuration(cfg.BlockRefreshInterval)
	refreshTimer := time.NewTimer(refreshIntv)
	Info.Printf("Set block refresh every %v", refreshIntv)

	purgeIntv := MustParseDuration(cfg.PurgeInterval)
	purgeTimer := time.NewTimer(purgeIntv)
	Info.Printf("Set purge interval to %v", purgeIntv)

	checkIntv, _ := time.ParseDuration(cfg.UpstreamCheckInterval)
	checkTimer := time.NewTimer(checkIntv)
	Info.Printf("Set check interval to %v", checkIntv)

	infoIntv, _ := time.ParseDuration(cfg.UpstreamCheckInterval)
	infoTimer := time.NewTimer(infoIntv)
	Info.Printf("Set info interval to %v", infoIntv)

	// Init block template
	go stratum.refreshBlockTemplate(false)

	go func() {
		for {
			select {
			case <-refreshTimer.C:
				stratum.refreshBlockTemplate(true)
				refreshTimer.Reset(refreshIntv)
			}
		}
	}()

	// purge stale
	go func() {
		for {
			select {
			case <-purgeTimer.C:
				stratum.purgeStale()
				purgeTimer.Reset(purgeIntv)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-checkTimer.C:
				stratum.checkUpstreams()
				checkTimer.Reset(checkIntv)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-infoTimer.C:
				poll := func(v *rpc.RPCClient) {
					_, err := v.UpdateInfo()
					if err != nil {
						Error.Printf("Unable to update info on upstream %s: %v", v.Name, err)
					}
				}
				current := stratum.rpc()
				poll(current)

				// Async rpc call to not block on rpc timeout, ignoring current
				go func() {
					for _, v := range stratum.upstreams {
						if v != current {
							poll(v)
						}
					}
				}()
				infoTimer.Reset(infoIntv)
			}
		}
	}()

	return stratum
}

func NewEndpoint(cfg *pool.Port) *Endpoint {
	e := &Endpoint{config: cfg}
	e.instanceId = make([]byte, 4)
	_, err := rand.Read(e.instanceId)
	if err != nil {
		Error.Fatalf("Can't seed with random bytes: %v", err)
	}
	e.targetHex = util.GetTargetHex(e.config.Difficulty)
	e.difficulty = big.NewInt(e.config.Difficulty)
	return e
}

func (s *StratumServer) Listen() {
	quit := make(chan bool)
	for _, port := range s.config.Stratum.Ports {
		go func(cfg pool.Port) {
			e := NewEndpoint(&cfg)
			e.Listen(s)
		}(port)
	}
	<-quit
}

func (e *Endpoint) Listen(s *StratumServer) {
	bindAddr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	addr, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		Error.Fatalf("Error: %v", err)
	}
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		Error.Fatalf("Error: %v", err)
	}
	defer server.Close()

	Info.Printf("Stratum listening on %s", bindAddr)
	accept := make(chan int, e.config.MaxConn)
	n := 0

	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			continue
		}
		_ = conn.SetKeepAlive(true)
		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		cs := &Session{conn: conn, ip: ip, enc: json.NewEncoder(conn), endpoint: e}
		n += 1

		accept <- n
		go func() {
			s.handleClient(cs, e)
			<-accept
		}()
	}
}

func (s *StratumServer) handleClient(cs *Session, e *Endpoint) {
	connbuff := bufio.NewReaderSize(cs.conn, MaxReqSize)
	s.setDeadline(cs.conn)

	for {
		data, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			Info.Println("Socket flood detected from", cs.ip)
			break
		} else if err == io.EOF {
			Info.Println("Client disconnected", cs.ip)
			break
		} else if err != nil {
			Error.Println("Error reading:", err)
			break
		}

		// NOTICE: cpuminer-multi sends junk newlines, so we demand at least 1 byte for decode
		// NOTICE: Ns*CNMiner.exe will send malformed JSON on very low diff, not sure we should handle this
		if len(data) > 1 {
			var req JSONRpcReq
			err = json.Unmarshal(data, &req)
			if err != nil {
				Error.Printf("Malformed request from %s: %v", cs.ip, err)
				break
			}
			s.setDeadline(cs.conn)
			err = cs.handleMessage(s, e, &req)
			if err != nil {
				break
			}
		}
	}
	s.removeSession(cs)
	_ = cs.conn.Close()
}

func (cs *Session) handleMessage(s *StratumServer, e *Endpoint, req *JSONRpcReq) error {
	if req.Id == nil {
		err := fmt.Errorf("server disconnect request")
		Error.Println(err)
		return err
	} else if req.Params == nil {
		err := fmt.Errorf("server RPC request params")
		Error.Println(err)
		return err
	}

	// Handle RPC methods
	switch req.Method {

	case "login":
		var params LoginParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			Error.Println("Unable to parse params")
			return err
		}
		reply, errReply := s.handleLoginRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, true)
		}
		return cs.sendResult(req.Id, &reply)
	case "getjob":
		var params GetJobParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			Error.Println("Unable to parse params")
			return err
		}
		reply, errReply := s.handleGetJobRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, true)
		}
		return cs.sendResult(req.Id, &reply)
	case "submit":
		var params SubmitParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			Error.Println("Unable to parse params")
			return err
		}
		reply, errReply := s.handleSubmitRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, false)
		}
		return cs.sendResult(req.Id, &reply)
	case "keepalived":
		return cs.sendResult(req.Id, &StatusReply{Status: "KEEPALIVED"})
	default:
		errReply := s.handleUnknownRPC(req)
		return cs.sendError(req.Id, errReply, true)
	}
}

func (cs *Session) sendResult(id *json.RawMessage, result interface{}) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONRpcResp{Id: id, Version: "2.0", Error: nil, Result: result}
	return cs.enc.Encode(&message)
}

func (cs *Session) pushMessage(method string, params interface{}) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONPushMessage{Version: "2.0", Method: method, Params: params}
	return cs.enc.Encode(&message)
}

func (cs *Session) sendError(id *json.RawMessage, reply *ErrorReply, drop bool) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONRpcResp{Id: id, Version: "2.0", Error: reply}
	err := cs.enc.Encode(&message)
	if err != nil {
		return err
	}
	if drop {
		return fmt.Errorf("server disconnect request")
	}
	return nil
}

func (s *StratumServer) setDeadline(conn *net.TCPConn) {
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
}

func (s *StratumServer) registerSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[cs] = struct{}{}
}

func (s *StratumServer) removeSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, cs)
}

func (s *StratumServer) isActive(cs *Session) bool {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	_, exist := s.sessions[cs]
	return exist
}

func (s *StratumServer) registerMiner(miner *Miner) {
	s.miners.Set(miner.id, miner)
}

func (s *StratumServer) removeMiner(id string) {
	s.miners.Remove(id)
}

func (s *StratumServer) currentBlockTemplate() *BlockTemplate {
	if t := s.blockTemplate.Load(); t != nil {
		return t.(*BlockTemplate)
	}
	return nil
}

func (s *StratumServer) checkUpstreams() {
	candidate := int32(0)
	backup := false

	for i, v := range s.upstreams {
		ok, err := v.Check(8, s.config.Address)
		if err != nil {
			Error.Printf("Upstream %v didn't pass check: %v", v.Name, err)
		}
		if ok && !backup {
			candidate = int32(i)
			backup = true
		}
	}

	if s.upstream != candidate {
		Info.Printf("Switching to %v upstream", s.upstreams[candidate].Name)
		atomic.StoreInt32(&s.upstream, candidate)
	}

	upstreamsAllSick := true
	for _, v := range s.upstreams {
		if !v.Sick() {
			upstreamsAllSick = false
			break
		}
	}

	s.upstreamsStates = append(s.upstreamsStates, upstreamsAllSick)
	if len(s.upstreamsStates) >= 60 {
		s.upstreamsStates = s.upstreamsStates[len(s.upstreamsStates)-60 : len(s.upstreamsStates)]

		// sick in the past 5 minutes, log and exit
		sickStatusAllTrue := true
		for _, v := range s.upstreamsStates {
			if !v {
				sickStatusAllTrue = false
				break
			}
		}

		if sickStatusAllTrue {
			Error.Fatal("all upstreams status were sick in the latest 5 minutes")
		}
	}
}

func (s *StratumServer) purgeStale() {
	start := time.Now()
	total, err := s.backend.FlushStaleStats(s.hashrateWindow, s.hashrateLargeWindow)
	if err != nil {
		Error.Println("Failed to purge stale data from backend:", err)
	} else {
		Info.Printf("Purged stale stats from backend, %v shares affected, elapsed time %v", total, time.Since(start))
	}
}

func (s *StratumServer) rpc() *rpc.RPCClient {
	i := atomic.LoadInt32(&s.upstream)
	return s.upstreams[i]
}

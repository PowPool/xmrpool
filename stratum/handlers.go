package stratum

import (
	"github.com/PowPool/xmrpool/cnutil"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/PowPool/xmrpool/util"
)

var noncePattern *regexp.Regexp

const defaultWorkerId = "0"

func init() {
	noncePattern, _ = regexp.Compile("^[0-9a-f]{8}$")
}

func (s *StratumServer) handleLoginRPC(cs *Session, params *LoginParams) (*JobReply, *ErrorReply) {
	address, id := extractWorkerId(params.Login)
	if !s.config.BypassAddressValidation && !cnutil.ValidateAddress(address) {
		Error.Printf("Invalid address %s used for login by %s", address, cs.ip)
		return nil, &ErrorReply{Code: -1, Message: "Invalid address used for login"}
	}

	t := s.currentBlockTemplate()
	if t == nil {
		return nil, &ErrorReply{Code: -1, Message: "Job not ready"}
	}

	cs.login = address
	cs.id = id

	miner, ok := s.miners.Get(cs.id)
	if !ok {
		miner = NewMiner(cs.id, cs.ip, s.maxConcurrency)
		s.registerMiner(miner)
	}

	Info.Printf("Miner connected %s.%s@%s", cs.login, cs.id, cs.ip)

	s.registerSession(cs)
	miner.heartbeat()

	return &JobReply{Id: cs.id, Job: cs.getJob(t), Status: "OK"}, nil
}

func (s *StratumServer) handleGetJobRPC(cs *Session, params *GetJobParams) (*JobReplyData, *ErrorReply) {
	miner, ok := s.miners.Get(params.Id)
	if !ok {
		return nil, &ErrorReply{Code: -1, Message: "Unauthenticated"}
	}
	t := s.currentBlockTemplate()
	if t == nil {
		return nil, &ErrorReply{Code: -1, Message: "Job not ready"}
	}
	miner.heartbeat()
	return cs.getJob(t), nil
}

func (s *StratumServer) handleSubmitRPC(cs *Session, params *SubmitParams) (*StatusReply, *ErrorReply) {
	miner, ok := s.miners.Get(params.Id)
	if !ok {
		return nil, &ErrorReply{Code: -1, Message: "Unauthenticated"}
	}
	miner.heartbeat()

	job := cs.findJob(params.JobId)
	if job == nil {
		return nil, &ErrorReply{Code: -1, Message: "Invalid job id"}
	}

	if !noncePattern.MatchString(params.Nonce) {
		return nil, &ErrorReply{Code: -1, Message: "Malformed nonce"}
	}
	nonce := strings.ToLower(params.Nonce)
	exist := job.submit(nonce)
	if exist {
		atomic.AddInt64(&miner.invalidShares, 1)
		return nil, &ErrorReply{Code: -1, Message: "Duplicate share"}
	}

	t := s.currentBlockTemplate()
	if job.height != t.height {
		Error.Printf("Stale share for height %d from %s.%s@%s", job.height, cs.login, cs.id, cs.ip)
		ShareLog.Printf("Stale share for height %d from %s.%s@%s", job.height, cs.login, cs.id, cs.ip)
		atomic.AddInt64(&miner.staleShares, 1)
		return nil, &ErrorReply{Code: -1, Message: "Block expired"}
	}

	validShare := miner.processShare(s, cs, job, t, nonce, params.Result, s.hashrateExpiration)
	if !validShare {
		return nil, &ErrorReply{Code: -1, Message: "Low difficulty share"}
	}
	return &StatusReply{Status: "OK"}, nil
}

func (s *StratumServer) handleUnknownRPC(req *JSONRpcReq) *ErrorReply {
	Error.Printf("Unknown RPC method: %v", req)
	return &ErrorReply{Code: -1, Message: "Invalid method"}
}

func (s *StratumServer) broadcastNewJobs() {
	t := s.currentBlockTemplate()
	if t == nil {
		return
	}
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	count := len(s.sessions)
	Info.Printf("Broadcasting new jobs to %d miners", count)

	start := time.Now()
	bcast := make(chan int, 1024)
	n := 0

	for m := range s.sessions {
		n++
		bcast <- n
		go func(cs *Session) {
			reply := cs.getJob(t)
			err := cs.pushMessage("job", &reply)
			<-bcast
			if err != nil {
				Error.Printf("Job transmit error to %s: %v", cs.ip, err)
				s.removeSession(cs)
			} else {
				if cs.tlsConn != nil {
					s.setTLSDeadline(cs.tlsConn)
				}
				if cs.conn != nil {
					s.setDeadline(cs.conn)
				}
			}
		}(m)
	}
	Info.Printf("Jobs broadcast finished %s", time.Since(start))
}

func (s *StratumServer) refreshBlockTemplate(bcast bool) {
	newBlock := s.fetchBlockTemplate()
	if newBlock && bcast {
		s.broadcastNewJobs()
	}
}

func extractWorkerId(loginWorkerPair string) (string, string) {
	parts := strings.SplitN(loginWorkerPair, ".", 2)
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return loginWorkerPair, defaultWorkerId
}

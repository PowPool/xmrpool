package stratum

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MiningPool0826/xmrpool/cnutil"
	"github.com/MiningPool0826/xmrpool/hashing"
	"github.com/MiningPool0826/xmrpool/util"
	. "github.com/MiningPool0826/xmrpool/util"
)

type Job struct {
	height int64
	sync.RWMutex
	id          string
	extraNonce  uint32
	submissions map[string]struct{}
}

type Miner struct {
	lastBeat      int64
	startedAt     int64
	validShares   int64
	invalidShares int64
	staleShares   int64
	accepts       int64
	rejects       int64
	shares        map[int64]int64
	sync.RWMutex
	id string
	ip string
}

func (job *Job) submit(nonce string) bool {
	job.Lock()
	defer job.Unlock()
	if _, exist := job.submissions[nonce]; exist {
		return true
	}
	job.submissions[nonce] = struct{}{}
	return false
}

func NewMiner(id string, ip string) *Miner {
	shares := make(map[int64]int64)
	return &Miner{id: id, ip: ip, shares: shares}
}

func (cs *Session) getJob(t *BlockTemplate) *JobReplyData {
	height := atomic.SwapInt64(&cs.lastBlockHeight, t.height)

	if height == t.height {
		return &JobReplyData{}
	}

	extraNonce := atomic.AddUint32(&cs.endpoint.extraNonce, 1)
	blob := t.nextBlob(extraNonce, cs.endpoint.instanceId)
	id := atomic.AddUint64(&cs.endpoint.jobSequence, 1)
	job := &Job{
		id:         strconv.FormatUint(id, 10),
		extraNonce: extraNonce,
		height:     t.height,
	}
	job.submissions = make(map[string]struct{})
	cs.pushJob(job)
	reply := &JobReplyData{JobId: job.id, Blob: blob, Target: cs.endpoint.targetHex}
	return reply
}

func (cs *Session) pushJob(job *Job) {
	cs.Lock()
	defer cs.Unlock()
	cs.validJobs = append(cs.validJobs, job)

	if len(cs.validJobs) > 4 {
		cs.validJobs = cs.validJobs[1:]
	}
}

func (cs *Session) findJob(id string) *Job {
	cs.Lock()
	defer cs.Unlock()
	for _, job := range cs.validJobs {
		if job.id == id {
			return job
		}
	}
	return nil
}

func (m *Miner) heartbeat() {
	now := util.MakeTimestamp()
	atomic.StoreInt64(&m.lastBeat, now)
}

func (m *Miner) getLastBeat() int64 {
	return atomic.LoadInt64(&m.lastBeat)
}

func (m *Miner) storeShare(diff int64) {
	now := util.MakeTimestamp() / 1000
	m.Lock()
	m.shares[now] += diff
	m.Unlock()
}

func (m *Miner) hashrate(estimationWindow time.Duration) float64 {
	now := util.MakeTimestamp() / 1000
	totalShares := int64(0)
	window := int64(estimationWindow / time.Second)
	boundary := now - m.startedAt

	if boundary > window {
		boundary = window
	}

	m.Lock()
	for k, v := range m.shares {
		if k < now-86400 {
			delete(m.shares, k)
		} else if k >= now-window {
			totalShares += v
		}
	}
	m.Unlock()
	return float64(totalShares) / float64(boundary)
}

func (m *Miner) processShare(s *StratumServer, cs *Session, job *Job, t *BlockTemplate, nonce string, result string) bool {
	r := s.rpc()

	shareBuff := make([]byte, len(t.buffer))
	copy(shareBuff, t.buffer)
	copy(shareBuff[t.reservedOffset+4:t.reservedOffset+7], cs.endpoint.instanceId)

	extraBuff := new(bytes.Buffer)
	binary.Write(extraBuff, binary.BigEndian, job.extraNonce)
	copy(shareBuff[t.reservedOffset:], extraBuff.Bytes())

	nonceBuff, _ := hex.DecodeString(nonce)
	copy(shareBuff[39:], nonceBuff)

	var hashBytes, convertedBlob []byte

	if s.config.BypassShareValidation {
		hashBytes, _ = hex.DecodeString(result)
	} else {
		convertedBlob = cnutil.ConvertBlob(shareBuff)
		hashBytes = hashing.Hash(convertedBlob, false, t.height)
	}

	if !s.config.BypassShareValidation && hex.EncodeToString(hashBytes) != result {
		Error.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		ShareLog.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}

	hashDiff, ok := util.GetHashDifficulty(hashBytes)
	if !ok {
		Error.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		ShareLog.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}
	block := hashDiff.Cmp(t.difficulty) >= 0

	nonceHex := hex.EncodeToString(nonceBuff)
	instanceIdHex := hex.EncodeToString(cs.endpoint.instanceId)
	extraHex := hex.EncodeToString(extraBuff.Bytes())
	paramIn := []string{nonceHex, instanceIdHex, extraHex}
	if block {
		_, err := r.SubmitBlock(hex.EncodeToString(shareBuff))
		if err != nil {
			atomic.AddInt64(&m.rejects, 1)
			atomic.AddInt64(&r.Rejects, 1)
			Error.Printf("Block rejected at height %d: %v", t.height, err)
			BlockLog.Printf("Block rejected at height %d: %v", t.height, err)
		} else {
			if len(convertedBlob) == 0 {
				convertedBlob = cnutil.ConvertBlob(shareBuff)
			}
			blockFastHash := hex.EncodeToString(hashing.FastHash(convertedBlob))
			now := util.MakeTimestamp()
			roundShares := atomic.SwapInt64(&s.roundShares, 0)
			ratio := float64(roundShares) / float64(t.diffInt64)
			s.blocksMu.Lock()
			s.blockStats[now] = blockEntry{height: t.height, hash: blockFastHash, variance: ratio}
			s.blocksMu.Unlock()
			atomic.AddInt64(&m.accepts, 1)
			atomic.AddInt64(&r.Accepts, 1)
			atomic.StoreInt64(&r.LastSubmissionAt, now)

			exist, err := s.backend.WriteBlock(cs.login, cs.id, paramIn, cs.endpoint.difficulty.Int64(), t.diffInt64, uint64(t.height),
				h.CoinBaseValue, h.JobTxsFeeTotal, s.hashrateExpiration)
			if exist {
				ms := MakeTimestamp()
				ts := ms / 1000

				err := s.backend.WriteInvalidShare(ms, ts, cs.login, cs.id, cs.endpoint.difficulty.Int64())
				if err != nil {
					Error.Println("Failed to insert invalid share data into backend:", err)
				}
				return false
			}
			if err != nil {
				Error.Println("Failed to insert block candidate into backend:", err)
				BlockLog.Println("Failed to insert block candidate into backend:", err)
			} else {
				Info.Printf("Inserted block %v to backend", t.height)
				BlockLog.Printf("Inserted block %v to backend", t.height)
			}

			Info.Printf("Block %s found at height %d by miner %v.%v@%v with ratio %.4f", blockFastHash[0:6], t.height, cs.login, cs.id, cs.ip, ratio)
			BlockLog.Printf("Block %s found at height %d by miner %v.%v@%v with ratio %.4f", blockFastHash[0:6], t.height, cs.login, cs.id, cs.ip, ratio)

			// Immediately refresh current BT and send new jobs
			s.refreshBlockTemplate(true)
		}
	} else if hashDiff.Cmp(cs.endpoint.difficulty) < 0 {
		ms := MakeTimestamp()
		ts := ms / 1000
		err := s.backend.WriteRejectShare(ms, ts, cs.login, cs.id, cs.endpoint.difficulty.Int64())
		if err != nil {
			Error.Println("Failed to insert reject share data into backend:", err)
			return false
		}
		Error.Printf("Rejected low difficulty share of %v from %v.%v@%v", hashDiff, cs.login, cs.id, cs.ip)
		ShareLog.Printf("Rejected low difficulty share of %v from %v.%v@%v", hashDiff, cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}

	atomic.AddInt64(&s.roundShares, cs.endpoint.config.Difficulty)
	atomic.AddInt64(&m.validShares, 1)
	m.storeShare(cs.endpoint.config.Difficulty)
	Info.Printf("Valid share at difficulty %v/%v", cs.endpoint.config.Difficulty, hashDiff)
	ShareLog.Printf("Valid share at difficulty %v/%v", cs.endpoint.config.Difficulty, hashDiff)
	return true
}

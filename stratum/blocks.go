package stratum

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/MiningPool0826/xmrpool/blocktemplate"
	"io"
	"math/big"

	"github.com/MiningPool0826/xmrpool/cnutil"
	. "github.com/MiningPool0826/xmrpool/util"
)

type BlockTemplate struct {
	diffInt64      int64
	height         int64
	difficulty     *big.Int
	reservedOffset int
	prevHash       string
	buffer         []byte

	blockReward int64
	txTotalFee  int64
}

func (b *BlockTemplate) nextBlob(extraNonce uint32, instanceId []byte) string {
	extraBuff := new(bytes.Buffer)
	binary.Write(extraBuff, binary.BigEndian, extraNonce)

	blobBuff := make([]byte, len(b.buffer))
	copy(blobBuff, b.buffer)
	copy(blobBuff[b.reservedOffset+4:b.reservedOffset+7], instanceId)
	copy(blobBuff[b.reservedOffset:], extraBuff.Bytes())
	blob := cnutil.ConvertBlob(blobBuff)
	return hex.EncodeToString(blob)
}

func (s *StratumServer) fetchBlockTemplate() bool {
	r := s.rpc()
	reply, err := r.GetBlockTemplate(8, s.config.Address)
	if err != nil {
		Error.Printf("Error while refreshing block template: %s", err)
		return false
	}
	t := s.currentBlockTemplate()

	if t != nil && t.prevHash == reply.PrevHash {
		// Fallback to height comparison
		if len(reply.PrevHash) == 0 && reply.Height > t.height {
			Info.Printf("New block to mine on %s at height %v, diff: %v", r.Name, reply.Height, reply.Difficulty)
		} else {
			return false
		}
	} else {
		Info.Printf("New block to mine on %s at height %v, diff: %v, prev_hash: %s", r.Name, reply.Height, reply.Difficulty, reply.PrevHash)
	}
	newTemplate := BlockTemplate{
		diffInt64:      reply.Difficulty,
		difficulty:     big.NewInt(reply.Difficulty),
		height:         reply.Height,
		prevHash:       reply.PrevHash,
		reservedOffset: reply.ReservedOffset,
	}
	newTemplate.buffer, _ = hex.DecodeString(reply.Blob)

	// set blockReward and txTotalFee
	var blockTemplateBlob blocktemplate.BlockTemplateBlob
	bytesBuf := bytes.NewBuffer(newTemplate.buffer)
	bufReader := io.Reader(bytesBuf)
	err = blockTemplateBlob.UnPack(bufReader)
	if err != nil {
		Error.Printf("unpack block template blob fail, blob hex string: %s", reply.Blob)
		return false
	}

	if len(blockTemplateBlob.Block.MinerTx.Vout) < 1 {
		Error.Printf("invalid block template blob (Vout count < 1), blob hex string: %s", reply.Blob)
		return false
	}
	newTemplate.blockReward = int64(blockTemplateBlob.Block.MinerTx.Vout[0].Amount)

	if reply.ExpectedReward < newTemplate.blockReward {
		Error.Printf("invalid block template blob (expectedReward: %d, blockReward: %d), blob hex string: %s",
			reply.ExpectedReward, newTemplate.blockReward, reply.Blob)
		return false
	}
	newTemplate.txTotalFee = reply.ExpectedReward - newTemplate.blockReward

	s.blockTemplate.Store(&newTemplate)
	return true
}

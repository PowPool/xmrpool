package blocktemplate

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
)

func TestBlockHashingBlob_Pack_UnPack(t *testing.T) {
	blkHashingBlobBytes, _ := hex.DecodeString("0e0eaed588840682dbba38bcf57bd78890057228be34a3b9115196f13a657228e53d05ee9a139f00000000a630518908e913764ea6b11886fc1629c67252b405b3130f9bbfb406be6f7b8e22")
	bytesBuf := bytes.NewBuffer(blkHashingBlobBytes)
	bufReader := io.Reader(bytesBuf)
	var blkHashingBlob BlockHashingBlob
	_ = blkHashingBlob.UnPack(bufReader)
	fmt.Println("tx hash count:", blkHashingBlob.TxHashSize)

	bytesBuf = bytes.NewBuffer([]byte{})
	bufWriter := io.Writer(bytesBuf)
	_ = blkHashingBlob.Pack(bufWriter)
	fmt.Println("block hashing blob hex:", hex.EncodeToString(bytesBuf.Bytes()))
}

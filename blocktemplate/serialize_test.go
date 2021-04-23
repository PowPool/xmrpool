package blocktemplate

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
)

func TestPackVarInt(t *testing.T) {
	bytesBuf := bytes.NewBuffer([]byte{})
	bufWriter := io.Writer(bytesBuf)
	_ = PackVarInt(bufWriter, 1618645622)
	fmt.Println("TestPackVarInt, hexStr:", hex.EncodeToString(bytesBuf.Bytes()))
}

func TestUnPackVarInt(t *testing.T) {
	bytesBuf := bytes.NewBuffer([]byte{0xf6, 0xa4, 0xea, 0x83, 0x06})
	bufReader := io.Reader(bytesBuf)
	varInt, _ := UnPackVarInt(bufReader)
	fmt.Println("TestUnPackVarInt, varInt:", varInt)
}

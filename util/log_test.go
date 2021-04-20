package util

import "testing"

func TestLog(t *testing.T) {
	InitLog("info.log", "error.log", "share.log", "block.log", 40)
	Debug.Println("debug")
	Error.Println("error")
	ShareLog.Println("share")
	BlockLog.Println("block")
}

package util

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
	"time"
)

var Xmr = math.BigPow(10, 12)
var Satoshi = math.BigPow(10, 0)

var Diff1 = StringToBig("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

func StringToBig(h string) *big.Int {
	n := new(big.Int)
	n.SetString(h, 0)
	return n
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func GetTargetHex(diff int64) string {
	padded := make([]byte, 32)

	diffBuff := new(big.Int).Div(Diff1, big.NewInt(diff)).Bytes()
	copy(padded[32-len(diffBuff):], diffBuff)
	buff := padded[0:4]
	targetHex := hex.EncodeToString(reverse(buff))
	return targetHex
}

func GetHashDifficulty(hashBytes []byte) (*big.Int, bool) {
	diff := new(big.Int)
	diff.SetBytes(reverse(hashBytes))

	// Check for broken result, empty string or zero hex value
	if diff.Cmp(new(big.Int)) == 0 {
		return nil, false
	}
	return diff.Div(Diff1, diff), true
}

//func ValidateAddress(addy string, poolAddy string) bool {
//	if len(addy) != len(poolAddy) {
//		return false
//	}
//	prefix, _ := utf8.DecodeRuneInString(addy)
//	poolPrefix, _ := utf8.DecodeRuneInString(poolAddy)
//	if prefix != poolPrefix {
//		return false
//	}
//	return cnutil.ValidateAddress(addy)
//}

func reverse(src []byte) []byte {
	dst := make([]byte, len(src))
	for i := len(src); i > 0; i-- {
		dst[len(src)-i] = src[i-1]
	}
	return dst
}

func MustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}

func FormatRatReward(reward *big.Rat) string {
	satoshi := new(big.Rat).SetInt(Xmr)
	reward = reward.Quo(reward, satoshi)
	return reward.FloatString(8)
}

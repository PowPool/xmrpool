package blocktemplate

import (
	"encoding/hex"
	"errors"
	"io"
)

const (
	CRYPTOHASHSIZE   = 32
	CRYPTOPUBKEYSIZE = 33
)

type CryptoHash struct {
	HashData [CRYPTOHASHSIZE]byte
}

func (c CryptoHash) Pack(writer io.Writer) error {
	_, err := writer.Write(c.HashData[:])
	if err != nil {
		return err
	}
	return nil
}

func (c *CryptoHash) UnPack(reader io.Reader) error {
	_, err := reader.Read(c.HashData[0:CRYPTOHASHSIZE])
	if err != nil {
		return err
	}
	return nil
}

func (c CryptoHash) ToHex() string {
	return hex.EncodeToString(c.HashData[:])
}

func (c *CryptoHash) FromHex(hexStr string) error {
	if len(hexStr) != 2*CRYPTOHASHSIZE {
		return errors.New("CryptoHash FromHex: invalid hex string len")
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}
	copy(c.HashData[0:], b[:])
	return nil
}

type CryptoPubKey struct {
	PubKeyData [CRYPTOPUBKEYSIZE]byte
}

func (c CryptoPubKey) Pack(writer io.Writer) error {
	_, err := writer.Write(c.PubKeyData[:])
	if err != nil {
		return err
	}
	return nil
}

func (c *CryptoPubKey) UnPack(reader io.Reader) error {
	_, err := reader.Read(c.PubKeyData[0:CRYPTOPUBKEYSIZE])
	if err != nil {
		return err
	}
	return nil
}

func (c CryptoPubKey) ToHex() string {
	return hex.EncodeToString(c.PubKeyData[:])
}

func (c *CryptoPubKey) FromHex(hexStr string) error {
	if len(hexStr) != 2*CRYPTOPUBKEYSIZE {
		return errors.New("CryptoPubKey FromHex: invalid hex string len")
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}
	copy(c.PubKeyData[0:], b[:])
	return nil
}

func PackVarInt(writer io.Writer, varInt uint64) error {
	var dest uint8 = 0
	for {
		if varInt < 0x80 {
			break
		} else {
			dest = uint8((varInt & 0x7f) | 0x80)
			varInt = varInt >> 7
		}
		_, err := writer.Write([]byte{dest})
		if err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte{uint8(varInt)})
	if err != nil {
		return err
	}
	return nil
}

func UnPackVarInt(reader io.Reader) (uint64, error) {
	var varIntBytes []byte
	for {
		var oneByte [1]byte
		_, err := reader.Read(oneByte[0:1])
		if err != nil {
			return 0, err
		}
		if oneByte[0] < 0x80 {
			varIntBytes = append(varIntBytes, oneByte[0]&0x7f)
			break
		} else {
			varIntBytes = append(varIntBytes, oneByte[0]&0x7f)
		}
	}
	var result uint64 = 0
	for i := 0; i < len(varIntBytes); i++ {
		result += uint64(varIntBytes[i]) << ((i) * 7)
	}
	return result, nil
}

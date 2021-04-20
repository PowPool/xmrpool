package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

func padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

func unpadding(src []byte) []byte {
	n := len(src)
	unPadNum := int(src[n-1])
	return src[:n-unPadNum]
}

func encryptAES(src []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	src = padding(src, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, key)
	blockMode.CryptBlocks(src, src)
	return src, nil
}

func decryptAES(src []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, key)
	blockMode.CryptBlocks(src, src)
	src = unpadding(src)
	return src, nil
}

func Ae64Encode(src []byte, key []byte) (string, error) {
	if len(key) <= 16 {
		paddingCount := 16 - len(key)
		for i := 0; i < paddingCount; i++ {
			key = append(key, byte(' '))
		}
	} else {
		key = key[0:16]
	}

	encryptBytes, err := encryptAES(src, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptBytes), nil
}

func Ae64Decode(str string, key []byte) ([]byte, error) {
	if len(key) <= 16 {
		paddingCount := 16 - len(key)
		for i := 0; i < paddingCount; i++ {
			key = append(key, byte(' '))
		}
	} else {
		key = key[0:16]
	}

	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	origBytes, err := decryptAES(decoded, key)
	if err != nil {
		return nil, err
	}
	return origBytes, nil
}

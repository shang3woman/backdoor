package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

var passwd = [16]byte{0x06, 0x00, 0xff, 0x07, 0x00, 0x90, 0x18, 0x71, 0x48, 0x15, 0x34, 0xa1, 0xb3, 0xc2, 0xd3, 0x81}
var iv = [16]byte{0x14, 0x23, 0x12, 0x11, 0x39, 0x89, 0x24, 0x00, 0x90, 0x18, 0x58, 0x69, 0x29, 0x18, 0xf1, 0xcb}

type Encoder struct {
	blockmode cipher.BlockMode
}

func NewEncoder() *Encoder {
	block, err := aes.NewCipher(passwd[:])
	if err != nil {
		panic(err)
	}
	pencoder := new(Encoder)
	pencoder.blockmode = cipher.NewCBCEncrypter(block, iv[:])
	return pencoder
}

func (e *Encoder) Encode(data []byte) []byte {
	newdata := pkcs7Padding(data, e.blockmode.BlockSize())
	result := make([]byte, len(newdata))
	e.blockmode.CryptBlocks(result, newdata)
	return result
}

type Decoder struct {
	blockmode cipher.BlockMode
}

func NewDecoder() *Decoder {
	block, err := aes.NewCipher(passwd[:])
	if err != nil {
		panic(err)
	}
	pdecoder := new(Decoder)
	pdecoder.blockmode = cipher.NewCBCDecrypter(block, iv[:])
	return pdecoder
}

func (d *Decoder) Decode(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%d.blockmode.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid length:%d", len(data))
	}
	result := make([]byte, len(data))
	d.blockmode.CryptBlocks(result, data)
	return unpkcs7Padding(result, d.blockmode.BlockSize())
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	var paddingText []byte
	if padding == 0 {
		paddingText = bytes.Repeat([]byte{byte(blockSize)}, blockSize)
	} else {
		paddingText = bytes.Repeat([]byte{byte(padding)}, padding)
	}
	tmp := make([]byte, 0, len(data)+len(paddingText))
	tmp = append(tmp, data...)
	tmp = append(tmp, paddingText...)
	return tmp
}

func unpkcs7Padding(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	unpadding := int(data[length-1])
	if unpadding == 0 || unpadding > blockSize {
		return nil, fmt.Errorf("unpkcs fail")
	}
	return data[:length-unpadding], nil
}

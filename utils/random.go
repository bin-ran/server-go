package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
)

func RandomBytes(len int) []byte {
	realLen := len / 4
	data := make([]uint32, realLen)
	for i := 0; i < realLen; i++ {
		data[i] = rand.Uint32()
	}

	bytesBuffer := bytes.NewBuffer([]byte{})

	binary.Write(bytesBuffer, binary.BigEndian, data)
	return bytesBuffer.Bytes()
}

func RandomURLBase64(len int) string {
	return base64.URLEncoding.EncodeToString(RandomBytes(len))
}

func RandomString(len int) string {
	result := make([]byte, len)
	var base int
	for i := 0; i < len; i++ {
		num := rand.Intn(62)
		if base < 10 {
			base = 48
		} else if base < 36 {
			base = 65 - 10
		} else {
			base = 97 - 36
		}

		result[i] = byte(base + num)
	}

	return string(result)
}

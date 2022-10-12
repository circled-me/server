package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"time"
)

// Sha512String hashes and encodes in hex the result
func Sha512String(s string) string {
	hash := sha512.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}

func Float32ArrayToByteArray(fa []float32) []byte {
	buf := bytes.Buffer{}
	_ = binary.Write(&buf, binary.LittleEndian, fa)
	return buf.Bytes()
}

func ByteArrayToFloat32Array(b []byte) (result []float32) {
	for i := 0; i < len(b); i += 4 {
		ui32 := uint32(b[i+0]) +
			uint32(b[i+1])<<8 +
			uint32(b[i+2])<<16 +
			uint32(b[i+3])<<24
		result = append(result, math.Float32frombits(ui32))
	}
	return
}

func GetDatesString(min, max int64) string {
	minString := time.Unix(min, 0).Format("2 Jan 2006")
	if max-min <= 86400 {
		return minString
	}
	maxString := time.Unix(max, 0).Format("2 Jan 2006")
	return minString + " - " + maxString
}

func Rand16BytesToBase62() string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		fmt.Println("error:", err)
		panic(err)
	}
	var i big.Int
	return i.SetBytes(buf).Text(62)
}

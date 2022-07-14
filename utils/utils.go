package utils

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"math"
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

package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/nfnt/resize"
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

func GetSeason(month time.Month, gpsLat *float64) string {
	if gpsLat == nil {
		return ""
	}
	if *gpsLat < 0 {
		month += 6 // add half an year for southern hemisphere
	}
	if month >= 3 && month <= 5 {
		return "Spring"
	} else if month >= 6 && month <= 8 {
		return "Summer"
	} else if month >= 9 && month <= 11 {
		return "Autumn/Fall"
	}
	return "Winter"
}

func GetDatesString(min, max int64) string {
	if min == 0 || max == 0 {
		return "empty :("
	}
	minString := time.Unix(min, 0).Format("2 Jan 2006")
	if max-min <= 86400 {
		return minString
	}
	maxString := time.Unix(max, 0).Format("2 Jan 2006")
	return minString + " - " + maxString
}

func RandSalt(saltSize int) string {
	b := make([]byte, saltSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func Rand16BytesToBase62() string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	var i big.Int
	return i.SetBytes(buf).Text(62)
}

func Rand8BytesToBase62() string {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	var i big.Int
	return i.SetBytes(buf).Text(62)
}

type ImageThumbConverted struct {
	ThumbSize int64
	NewX      uint16
	NewY      uint16
	OldX      uint16
	OldY      uint16
}

func CreateThumb(size uint, reader io.Reader, writer io.Writer) (result ImageThumbConverted, err error) {
	image, _, err := image.Decode(reader)
	if err != nil {
		return result, err
	}
	var newBuf bytes.Buffer
	newImage := resize.Thumbnail(size, size, image, resize.Lanczos3)
	if err = jpeg.Encode(&newBuf, newImage, &jpeg.Options{Quality: 90}); err != nil {
		return
	}
	imageRect := newImage.Bounds().Size()
	result.NewX = uint16(imageRect.X)
	result.NewY = uint16(imageRect.Y)

	imageRect = image.Bounds().Size()
	result.OldX = uint16(imageRect.X)
	result.OldY = uint16(imageRect.Y)

	result.ThumbSize, err = io.Copy(writer, &newBuf)
	return
}

func StringToFloat64Ptr(in string) *float64 {
	f, _ := strconv.ParseFloat(in, 64)
	return &f
}

func StringToUInt16(in string) uint16 {
	i, _ := strconv.ParseUint(in, 10, 16)
	return uint16(i)
}

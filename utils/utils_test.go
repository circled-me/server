package utils

import (
	"testing"
)

func TestFloat32ArrayToByteArray(t *testing.T) {
	fa := []float32{0.1, 0.2, 0.3}
	t.Errorf("%v", ByteArrayToFloat32Array(Float32ArrayToByteArray(fa)))
}

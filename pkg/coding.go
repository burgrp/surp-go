package surp

import (
	"encoding/binary"
	"math"
)

func EncodeString(v string) []byte {
	return []byte(v)
}

func DecodeString(b []byte) string {
	return string(b)
}

func EncodeInt(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func DecodeInt(b []byte) int64 {
	if len(b) != 8 {
		return 0
	}
	result := binary.BigEndian.Uint64(b)
	return int64(result)
}

func EncodeBool(v bool) []byte {
	if v {
		return []byte{1}
	}
	return []byte{0}
}

func DecodeBool(b []byte) bool {
	return len(b) == 1 && b[0] != 0
}

func EncodeFloat(v float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(v))
	return buf
}

func DecodeFloat(b []byte) float64 {
	if len(b) != 8 {
		return 0
	}
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits)
}

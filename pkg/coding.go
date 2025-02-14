package surp

import (
	"encoding/binary"
	"math"
)

func EncodeString(v string) []byte {
	return []byte(v)
}

func DecodeString(b []byte) (string, bool) {
	return string(b), true
}

func EncodeInt(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func DecodeInt(b []byte) (int64, bool) {
	if len(b) != 8 {
		return 0, false
	}
	result := binary.BigEndian.Uint64(b)
	return int64(result), true
}

func EncodeBool(v bool) []byte {
	if v {
		return []byte{1}
	}
	return []byte{0}
}

func DecodeBool(b []byte) (bool, bool) {
	if len(b) != 1 {
		return false, false
	}
	return b[0] != 0, true
}

func EncodeFloat(v float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(v))
	return buf
}

func DecodeFloat(b []byte) (float64, bool) {
	if len(b) != 8 {
		return 0, false
	}
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits), true
}

package surp

import (
	"encoding/binary"
)

func EncodeString(v string) []byte {
	return []byte(v)
}

func DecodeString(b []byte) string {
	return string(b)
}

func EncodeInt(v int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func DecodeInt(b []byte) int {
	result := binary.BigEndian.Uint64(b)
	return int(result)
}

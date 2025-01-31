package surp

import (
	"encoding/binary"
)

func encodeString(v string) []byte {
	return []byte(v)
}

func decodeString(b []byte) string {
	return string(b)
}

func encodeInt(v int) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func decodeInt(b []byte) int {
	result := binary.BigEndian.Uint64(b)
	return int(result)
}

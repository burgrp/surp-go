package surp

import (
	"bytes"
	"encoding/binary"
)

const magicString = "SURP"

type Message struct {
	SequenceNumber uint16
	Type           byte
	Group          string
	Name           string
	Value          Optional[[]byte]
	Metadata       map[string]string
}

func encodeMessage(msg *Message) []byte {
	var buf bytes.Buffer

	buf.WriteString(magicString)
	buf.WriteByte(msg.Type)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.Group)))
	buf.WriteString(msg.Group)
	buf.WriteByte(byte(len(msg.Name)))
	buf.WriteString(msg.Name)

	if msg.Type == MessageTypeSync || msg.Type == MessageTypeSet {

		writeValue(msg.Value, &buf)

		if msg.Type == MessageTypeSync {

			buf.WriteByte(byte(len(msg.Metadata)))
			for k, v := range msg.Metadata {
				buf.WriteByte(byte(len(k)))
				buf.WriteString(k)
				buf.WriteByte(byte(len(v)))
				buf.WriteString(v)
			}
		}
	}
	return buf.Bytes()
}

func writeValue(value Optional[[]byte], buf *bytes.Buffer) {
	var length int
	var data []byte

	if value.IsDefined() {
		data = value.Get()
		length = len(data)
	} else {
		length = -1
	}
	binary.Write(buf, binary.BigEndian, uint16(length))
	buf.Write(data)
}

func readByte(remaining *[]byte) (byte, bool) {
	if len(*remaining) < 1 {
		return 0, false
	}
	result := (*remaining)[0]
	*remaining = (*remaining)[1:]
	return result, true
}

func readUint16(remaining *[]byte) (uint16, bool) {
	if len(*remaining) < 2 {
		return 0, false
	}
	result := binary.BigEndian.Uint16((*remaining)[:2])
	*remaining = (*remaining)[2:]
	return result, true
}

func readString(remaining *[]byte) (string, bool) {
	if len(*remaining) < 1 {
		return "", false
	}
	length := int((*remaining)[0])
	if len(*remaining) < length+1 {
		return "", false
	}
	result := string((*remaining)[1 : length+1])
	*remaining = (*remaining)[length+1:]
	return result, true
}

func readValue(remaining *[]byte) (Optional[[]byte], bool) {

	value := NewUndefined[[]byte]()

	valueLen, ok := readUint16(remaining)
	if !ok {
		return value, false
	}

	if valueLen == 0xFFFF {
		return value, true
	}

	if len(*remaining) < int(valueLen) {
		return value, false
	}
	value = NewDefined((*remaining)[:valueLen])
	*remaining = (*remaining)[valueLen:]
	return value, true
}

func decodeMessage(data []byte) (*Message, bool) {

	remaining := data[:]

	msg := &Message{}

	var ok bool
	msg.Type, ok = readByte(&remaining)
	if !ok {
		return nil, false
	}

	if msg.Type != MessageTypeGet && msg.Type != MessageTypeSet && msg.Type != MessageTypeSync {
		return nil, false
	}

	msg.SequenceNumber, ok = readUint16(&remaining)
	if !ok {
		return nil, false
	}

	msg.Group, ok = readString(&remaining)
	if !ok {
		return nil, false
	}

	msg.Name, ok = readString(&remaining)
	if !ok {
		return nil, false
	}

	if msg.Type == MessageTypeSync || msg.Type == MessageTypeSet {

		msg.Value, ok = readValue(&remaining)
		if !ok {
			return nil, false
		}

		if msg.Type == MessageTypeSync {

			metadataCount, ok := readByte(&remaining)
			if !ok {
				return nil, false
			}

			msg.Metadata = make(map[string]string, metadataCount)

			for j := 0; j < int(metadataCount); j++ {

				key, ok := readString(&remaining)
				if !ok {
					return nil, false
				}

				val, ok := readString(&remaining)
				if !ok {
					return nil, false
				}

				msg.Metadata[key] = val
			}

		}

	}

	return msg, true
}

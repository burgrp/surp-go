package surp

import (
	"bytes"
	"encoding/binary"
)

const magicString = "SURP"

type AdvertisedRegister struct {
	Name     string
	Value    Optional[[]byte]
	Metadata map[string]string
}

type AdvertiseMessage struct {
	SequenceNumber uint16
	GroupName      string
	Port           uint16
	Registers      []AdvertisedRegister
}

type UpdateMessage struct {
	SequenceNumber uint16
	GroupName      string
	RegisterName   string
	Value          Optional[[]byte]
}

type SetMessage UpdateMessage

func encodeAdvertiseMessage(msg *AdvertiseMessage) []byte {
	var buf bytes.Buffer

	buf.WriteString(magicString)
	buf.WriteByte(messageTypeAdvertise)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.GroupName)))
	buf.WriteString(msg.GroupName)
	binary.Write(&buf, binary.BigEndian, msg.Port)

	binary.Write(&buf, binary.BigEndian, uint16(len(msg.Registers)))
	for _, r := range msg.Registers {
		buf.WriteByte(byte(len(r.Name)))
		buf.WriteString(r.Name)
		writeValue(r.Value, &buf)
		buf.WriteByte(byte(len(r.Metadata)))
		for k, v := range r.Metadata {
			buf.WriteByte(byte(len(k)))
			buf.WriteString(k)
			buf.WriteByte(byte(len(v)))
			buf.WriteString(v)
		}
	}

	return buf.Bytes()
}

func writeValue(value Optional[[]byte], buf *bytes.Buffer) {
	var length int
	var data []byte

	if value.IsValid() {
		data = value.Get()
		length = len(data)
	} else {
		length = -1
	}
	binary.Write(buf, binary.BigEndian, uint16(length))
	buf.Write(data)
}

func decodeAdvertiseMessage(data []byte) (*AdvertiseMessage, bool) {

	remaining := data[:]

	msg := &AdvertiseMessage{}

	if len(remaining) < 2 {
		return nil, false
	}
	msg.SequenceNumber = binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	if len(remaining) < 1 {
		return nil, false
	}
	groupNameLen := int(remaining[0])
	remaining = remaining[1:]
	if len(remaining) < groupNameLen {
		return nil, false
	}
	msg.GroupName = string(remaining[:groupNameLen])
	remaining = remaining[groupNameLen:]

	if len(remaining) < 2 {
		return nil, false
	}
	msg.Port = binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	registerCount := binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	for i := 0; i < int(registerCount); i++ {
		if len(remaining) < 1 {
			return nil, false
		}
		registerNameLen := int(remaining[0])
		remaining = remaining[1:]
		if len(remaining) < registerNameLen {
			return nil, false
		}
		registerName := string(remaining[:registerNameLen])
		remaining = remaining[registerNameLen:]

		value, remaining, ok := readValue(remaining)
		if !ok {
			return nil, false
		}

		if len(remaining) < 2 {
			return nil, false
		}
		metadataCount := remaining[0]
		remaining = remaining[1:]
		metadata := make(map[string]string, metadataCount)

		for j := 0; j < int(metadataCount); j++ {
			if len(remaining) < 1 {
				return nil, false
			}
			keyLen := int(remaining[0])
			remaining = remaining[1:]
			if len(remaining) < keyLen {
				return nil, false
			}
			k := string(remaining[:keyLen])
			remaining = remaining[keyLen:]

			if len(remaining) < 1 {
				return nil, false
			}
			valLen := int(remaining[0])
			remaining = remaining[1:]
			if len(remaining) < valLen {
				return nil, false
			}
			v := string(remaining[:valLen])
			remaining = remaining[valLen:]
			metadata[k] = v
		}

		msg.Registers = append(msg.Registers, AdvertisedRegister{
			Name:     registerName,
			Value:    value,
			Metadata: metadata,
		})
	}

	return msg, true
}

func readValue(remaining []byte) (Optional[[]byte], []byte, bool) {
	if len(remaining) < 2 {
		return NewInvalid[[]byte](), nil, false
	}
	valueLen := int16(binary.BigEndian.Uint16(remaining[:2]))
	remaining = remaining[2:]
	if valueLen == -1 {
		return NewInvalid[[]byte](), remaining, true
	}

	if len(remaining) < int(valueLen) {
		return NewInvalid[[]byte](), nil, false
	}
	value := remaining[:valueLen]
	remaining = remaining[valueLen:]
	return NewValid(value), remaining, true
}

func encodeValueMessage(messageType byte, msg *UpdateMessage) []byte {
	var buf bytes.Buffer

	buf.WriteString(magicString)
	buf.WriteByte(messageType)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.GroupName)))
	buf.WriteString(msg.GroupName)
	binary.Write(&buf, binary.BigEndian, uint16(len(msg.RegisterName)))
	buf.WriteString(msg.RegisterName)
	writeValue(msg.Value, &buf)

	return buf.Bytes()
}

func decodeValueMessage(messageType byte, data []byte) (*UpdateMessage, bool) {

	remaining := data[:]

	msg := &UpdateMessage{}

	if len(remaining) < 2 {
		return nil, false
	}
	msg.SequenceNumber = binary.BigEndian.Uint16(remaining[:2])
	remaining = remaining[2:]

	if len(remaining) < 1 {
		return nil, false
	}
	groupNameLen := int(remaining[0])
	remaining = remaining[1:]
	if len(remaining) < groupNameLen {
		return nil, false
	}
	msg.GroupName = string(remaining[:groupNameLen])
	remaining = remaining[groupNameLen:]

	if len(remaining) < 2 {
		return nil, false
	}
	registerNameLen := int(binary.BigEndian.Uint16(remaining[:2]))
	remaining = remaining[2:]
	if len(remaining) < registerNameLen {
		return nil, false
	}
	msg.RegisterName = string(remaining[:registerNameLen])
	remaining = remaining[registerNameLen:]

	value, _, ok := readValue(remaining)
	if !ok {
		return nil, false
	}
	msg.Value = value

	return msg, true
}

func encodeUpdateMessage(msg *UpdateMessage) []byte {
	return encodeValueMessage(messageTypeUpdate, (*UpdateMessage)(msg))
}

func decodeUpdateMessage(data []byte) (*UpdateMessage, bool) {
	msg, ok := decodeValueMessage(messageTypeUpdate, data)
	return msg, ok
}

func encodeSetMessage(msg *SetMessage) []byte {
	return encodeValueMessage(messageTypeSet, (*UpdateMessage)(msg))
}

func decodeSetMessage(data []byte) (*SetMessage, bool) {
	msg, ok := decodeValueMessage(messageTypeSet, data)
	return (*SetMessage)(msg), ok
}

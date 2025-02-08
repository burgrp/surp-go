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

type UpdatedRegister struct {
	Name  string
	Value Optional[[]byte]
}

type UpdateMessage struct {
	SequenceNumber uint16
	GroupName      string
	Registers      []UpdatedRegister
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

	buf.WriteByte(byte(len(msg.Registers)))
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

func readByte(data []byte) (byte, []byte, bool) {
	if len(data) < 1 {
		return 0, nil, false
	}
	return data[0], data[1:], true
}

func readUint16(data []byte) (uint16, []byte, bool) {
	if len(data) < 2 {
		return 0, nil, false
	}
	return binary.BigEndian.Uint16(data[:2]), data[2:], true
}

func readString(data []byte) (string, []byte, bool) {
	if len(data) < 1 {
		return "", nil, false
	}
	length := int(data[0])
	if len(data) < length+1 {
		return "", nil, false
	}
	return string(data[1 : length+1]), data[length+1:], true
}

func decodeAdvertiseMessage(data []byte) (*AdvertiseMessage, bool) {

	remaining := data[:]

	msg := &AdvertiseMessage{}

	sequenceNumber, remaining, ok := readUint16(remaining)
	if !ok {
		return nil, false
	}
	msg.SequenceNumber = sequenceNumber

	groupName, remaining, ok := readString(remaining)
	if !ok {
		return nil, false
	}
	msg.GroupName = groupName

	port, remaining, ok := readUint16(remaining)
	if !ok {
		return nil, false
	}
	msg.Port = port

	registerCount, remaining, ok := readByte(remaining)
	if !ok {
		return nil, false
	}

	msg.Registers = make([]AdvertisedRegister, 0, registerCount)

	for i := 0; i < int(registerCount); i++ {

		registerName, remaining, ok := readString(remaining)
		if !ok {
			return nil, false
		}

		value, remaining, ok := readValue(remaining)
		if !ok {
			return nil, false
		}

		metadataCount, remaining, ok := readByte(remaining)
		if !ok {
			return nil, false
		}

		metadata := make(map[string]string, metadataCount)

		for j := 0; j < int(metadataCount); j++ {

			var key string
			key, remaining, ok = readString(remaining)
			if !ok {
				return nil, false
			}

			var val string
			val, remaining, ok = readString(remaining)
			if !ok {
				return nil, false
			}

			metadata[key] = val
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
	buf.WriteByte(byte(len(msg.Registers)))
	for _, register := range msg.Registers {
		buf.WriteByte(byte(len(register.Name)))
		buf.WriteString(register.Name)
		writeValue(register.Value, &buf)
	}

	return buf.Bytes()
}

func decodeValueMessage(messageType byte, data []byte) (*UpdateMessage, bool) {

	remaining := data[:]

	msg := &UpdateMessage{}

	sequenceNumber, remaining, ok := readUint16(remaining)
	if !ok {
		return nil, false
	}
	msg.SequenceNumber = sequenceNumber

	groupName, remaining, ok := readString(remaining)
	if !ok {
		return nil, false
	}
	msg.GroupName = groupName

	registerCount, remaining, ok := readByte(remaining)
	if !ok {
		return nil, false
	}

	msg.Registers = make([]UpdatedRegister, 0, registerCount)

	for i := 0; i < int(registerCount); i++ {

		var name string
		name, remaining, ok = readString(remaining)
		if !ok {
			return nil, false
		}

		var value Optional[[]byte]
		value, remaining, ok = readValue(remaining)
		if !ok {
			return nil, false
		}

		msg.Registers = append(msg.Registers, UpdatedRegister{
			Name:  name,
			Value: value,
		})

	}

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

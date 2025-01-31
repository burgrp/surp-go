package surp

import (
	"bytes"
	"encoding/binary"
)

type AdvertisedRegister struct {
	RegisterName string
	Value        []byte
	Metadata     map[string]string
}

type AdvertiseMessage struct {
	SequenceNumber uint16
	GroupName      string
	Registers      []AdvertisedRegister
}

type UpdateMessage struct {
	SequenceNumber uint16
	GroupName      string
	RegisterName   string
	Value          []byte
}

func encodeAdvertiseMessage(msg *AdvertiseMessage) []byte {
	var buf bytes.Buffer

	buf.WriteByte(messageTypeAdvertise)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.GroupName)))
	buf.WriteString(msg.GroupName)
	binary.Write(&buf, binary.BigEndian, uint16(len(msg.Registers)))

	for _, r := range msg.Registers {
		buf.WriteByte(byte(len(r.RegisterName)))
		buf.WriteString(r.RegisterName)
		binary.Write(&buf, binary.BigEndian, uint16(len(r.Value)))
		buf.Write(r.Value)
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

func decodeAdvertiseMessage(data []byte) (*AdvertiseMessage, bool) {

	remaining := data[:]

	if len(remaining) < 1 {
		return nil, false
	}
	if remaining[0] != messageTypeAdvertise {
		return nil, false
	}
	remaining = remaining[1:]

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

		if len(remaining) < 2 {
			return nil, false
		}
		valueLen := binary.BigEndian.Uint16(remaining[:2])
		remaining = remaining[2:]
		if len(remaining) < int(valueLen) {
			return nil, false
		}
		value := remaining[:valueLen]
		remaining = remaining[valueLen:]

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
			RegisterName: registerName,
			Value:        value,
			Metadata:     metadata,
		})
	}

	return msg, true
}

func encodeUpdateMessage(msg *UpdateMessage) []byte {
	var buf bytes.Buffer

	buf.WriteByte(messageTypeUpdate)
	binary.Write(&buf, binary.BigEndian, msg.SequenceNumber)
	buf.WriteByte(byte(len(msg.GroupName)))
	buf.WriteString(msg.GroupName)
	binary.Write(&buf, binary.BigEndian, uint16(len(msg.RegisterName)))
	buf.WriteString(msg.RegisterName)
	binary.Write(&buf, binary.BigEndian, uint16(len(msg.Value)))
	buf.Write(msg.Value)

	return buf.Bytes()
}

func decodeUpdateMessage(data []byte) (*UpdateMessage, bool) {

	remaining := data[:]

	if len(remaining) < 1 {
		return nil, false
	}
	if remaining[0] != messageTypeUpdate {
		return nil, false
	}
	remaining = remaining[1:]

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

	if len(remaining) < 2 {
		return nil, false
	}
	valueLen := int(binary.BigEndian.Uint16(remaining[:2]))
	remaining = remaining[2:]
	if len(remaining) < valueLen {
		return nil, false
	}
	msg.Value = remaining[:valueLen]

	return msg, true
}

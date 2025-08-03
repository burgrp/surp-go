package surp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

func encodeValue(buf *bytes.Buffer, vt ValueType, value any) error {
	switch vt {
	case ValueBool:
		b := byte(0x00)
		if value.(bool) {
			b = 0x01
		}
		buf.WriteByte(b)
	case ValueU8, ValueS8:
		buf.WriteByte(value.(byte))
	case ValueU16, ValueS16:
		return binary.Write(buf, binary.BigEndian, value.(uint16))
	case ValueU32, ValueS32:
		return binary.Write(buf, binary.BigEndian, value.(uint32))
	case ValueU64, ValueS64:
		return binary.Write(buf, binary.BigEndian, value.(uint64))
	case ValueFloat64:
		return binary.Write(buf, binary.BigEndian, math.Float64bits(value.(float64)))
	case ValueShortString:
		s := []byte(value.(string))
		buf.WriteByte(byte(len(s)))
		buf.Write(s)
	case ValueLongString:
		s := []byte(value.(string))
		binary.Write(buf, binary.BigEndian, uint16(len(s)))
		buf.Write(s)
	default:
		return nil // UNDEFINED, or unsupported
	}
	return nil
}

func decodeValue(vt ValueType, buf *bytes.Reader) (any, error) {
	switch vt {
	case ValueBool:
		b, _ := buf.ReadByte()
		return b != 0, nil
	case ValueU8, ValueS8:
		return buf.ReadByte()
	case ValueU16, ValueS16:
		var v uint16
		err := binary.Read(buf, binary.BigEndian, &v)
		return v, err
	case ValueU32, ValueS32:
		var v uint32
		err := binary.Read(buf, binary.BigEndian, &v)
		return v, err
	case ValueU64, ValueS64:
		var v uint64
		err := binary.Read(buf, binary.BigEndian, &v)
		return v, err
	case ValueFloat64:
		var bits uint64
		err := binary.Read(buf, binary.BigEndian, &bits)
		return math.Float64frombits(bits), err
	case ValueShortString:
		length, _ := buf.ReadByte()
		str := make([]byte, length)
		_, err := buf.Read(str)
		return string(str), err
	case ValueLongString:
		var length uint16
		binary.Read(buf, binary.BigEndian, &length)
		str := make([]byte, length)
		_, err := buf.Read(str)
		return string(str), err
	default:
		return nil, nil
	}
}

func safeRead(r *bytes.Reader, buf []byte) error {
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return errors.New("unexpected end of buffer")
	}
	return nil
}

func DecodeMessageHeader(data []byte) (version uint8, msgType MsgType, err error) {
	if len(data) < 2 {
		return 0, 0, errors.New("packet too short for header")
	}
	version = data[0]
	msgType = MsgType(data[1])
	return version, msgType, nil
}

func EncodeMessageIS(msg *MessageIS) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(ProtocolVersion1) // Version
	buf.WriteByte(byte(MsgTypeIS))  // MsgType
	binary.Write(buf, binary.BigEndian, msg.TTL)
	buf.WriteByte(byte(len(msg.Name)))
	buf.Write([]byte(msg.Name))
	buf.WriteByte(byte(msg.ValueType))

	if msg.ValueType != ValueUndefined {
		if err := encodeValue(buf, msg.ValueType, msg.Value); err != nil {
			return nil, err
		}
	}

	buf.WriteByte(byte(len(msg.Metadata)))
	for _, meta := range msg.Metadata {
		buf.WriteByte(byte(meta.Key))
		buf.WriteByte(byte(meta.ValueType))
		if err := encodeValue(buf, meta.ValueType, meta.Value); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DecodeMessageIS(data []byte) (*MessageIS, error) {
	buf := bytes.NewReader(data)

	var ttl uint16
	if err := binary.Read(buf, binary.BigEndian, &ttl); err != nil {
		return nil, err
	}

	nameLen, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLen)
	if err := safeRead(buf, nameBytes); err != nil {
		return nil, err
	}
	name := string(nameBytes)

	vtByte, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	vt := ValueType(vtByte)

	var value any
	if vt != ValueUndefined {
		value, err = decodeValue(vt, buf)
		if err != nil {
			return nil, err
		}
	}

	metaCount, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	var metadata []MetadataEntry
	for i := 0; i < int(metaCount); i++ {
		key, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		mvtByte, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		mvt := ValueType(mvtByte)
		val, err := decodeValue(mvt, buf)
		if err != nil {
			return nil, err
		}
		metadata = append(metadata, MetadataEntry{
			Key:       MetadataKey(key),
			ValueType: mvt,
			Value:     val,
		})
	}

	return &MessageIS{
		TTL:       ttl,
		Name:      name,
		ValueType: vt,
		Value:     value,
		Metadata:  metadata,
	}, nil
}

func EncodeMessageSET(msg *MessageSET) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(ProtocolVersion1)
	buf.WriteByte(byte(MsgTypeSET))
	buf.WriteByte(byte(len(msg.Name)))
	buf.Write([]byte(msg.Name))
	buf.WriteByte(byte(msg.ValueType))
	if err := encodeValue(buf, msg.ValueType, msg.Value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeMessageSET(data []byte) (*MessageSET, error) {
	buf := bytes.NewReader(data)

	nameLen, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLen)
	if err := safeRead(buf, nameBytes); err != nil {
		return nil, err
	}
	name := string(nameBytes)

	vtByte, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	vt := ValueType(vtByte)

	val, err := decodeValue(vt, buf)
	if err != nil {
		return nil, err
	}

	return &MessageSET{
		Name:      name,
		ValueType: vt,
		Value:     val,
	}, nil
}

func EncodeMessageGET(msg *MessageGET) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(ProtocolVersion1)
	buf.WriteByte(byte(MsgTypeGET))
	binary.Write(buf, binary.BigEndian, msg.TTL)
	buf.WriteByte(byte(len(msg.Name)))
	buf.Write([]byte(msg.Name))
	return buf.Bytes(), nil
}

func DecodeMessageGET(data []byte) (*MessageGET, error) {
	buf := bytes.NewReader(data)

	var ttl uint16
	if err := binary.Read(buf, binary.BigEndian, &ttl); err != nil {
		return nil, err
	}

	nameLen, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLen)
	if err := safeRead(buf, nameBytes); err != nil {
		return nil, err
	}
	name := string(nameBytes)

	return &MessageGET{
		TTL:  ttl,
		Name: name,
	}, nil
}

package surp

import (
	"fmt"
	"strconv"
	"strings"
)

func StringToType(s string) (ValueType, error) {
	for typ, name := range VALUE_TYPE_NAMES {
		if name == s {
			return typ, nil
		}
	}
	return 0, fmt.Errorf("unknown value type %q, valid types are: %v", s, VALUE_TYPE_NAMES)
}

func TypeToString(t ValueType) string {
	return VALUE_TYPE_NAMES[t]
}

// ParseString parses a string to a value of the specified type.
func ParseString(valueStr string, typ ValueType) (any, error) {

	var value any
	var err error

	switch typ {
	case ValueBool:
		value = strings.ToUpper(valueStr) == "TRUE" || valueStr == "1"
	case ValueS8, ValueU8, ValueS16, ValueU16, ValueS32, ValueU32, ValueS64, ValueU64:
		var valueInt64 int64
		valueInt64, err = strconv.ParseInt(valueStr, 10, 8)
		switch typ {
		case ValueS8:
			value = int8(valueInt64)
		case ValueU8:
			value = uint8(valueInt64)
		case ValueS16:
			value = int16(valueInt64)
		case ValueU16:
			value = uint16(valueInt64)
		case ValueS32:
			value = int32(valueInt64)
		case ValueU32:
			value = uint32(valueInt64)
		case ValueS64:
			value = int64(valueInt64)
		case ValueU64:
			value = uint64(valueInt64)
		}
	case ValueFloat64:
		value, err = strconv.ParseFloat(valueStr, 64)
	case ValueShortString, ValueLongString:
		value = valueStr
	}

	return value, err
}

func StringToMetadataKey(s string) (MetadataKey, error) {
	for key, name := range METADATA_KEY_NAMES {
		if name == s {
			return key, nil
		}
	}
	return 0, fmt.Errorf("unknown metadata key %q, valid keys are: %v", s, METADATA_KEY_NAMES)
}

func MetadataKeyToString(key MetadataKey) string {
	return METADATA_KEY_NAMES[key]
}

func GetMetadataValue(metadata []MetadataEntry, key MetadataKey) (any, ValueType) {
	for _, entry := range metadata {
		if entry.Key == key {
			return entry.Value, entry.ValueType
		}
	}
	return nil, ValueUndefined
}

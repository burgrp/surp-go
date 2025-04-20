package surp

import (
	"fmt"
	"strconv"
	"strings"
)

func StringToType(s string) (ValueType, error) {
	switch s {
	case "bool":
		return ValueBool, nil
	case "s8":
		return ValueS8, nil
	case "u8":
		return ValueU8, nil
	case "s16":
		return ValueS16, nil
	case "u16":
		return ValueU16, nil
	case "s32":
		return ValueS32, nil
	case "u32":
		return ValueU32, nil
	case "s64":
		return ValueS64, nil
	case "u64":
		return ValueU64, nil
	case "dbl":
		return ValueDouble, nil
	case "ss":
		return ValueShortString, nil
	case "ls":
		return ValueLongString, nil
	default:
		return 0, fmt.Errorf("type %q is not one of supported types: bool, s8, u8, s16, u16, s32, u32, s64, u64, dbl, ss, ls", s)
	}
}

func TypeToString(t ValueType) string {
	switch t {
	case ValueBool:
		return "bool"
	case ValueS8:
		return "s8"
	case ValueU8:
		return "u8"
	case ValueS16:
		return "s16"
	case ValueU16:
		return "u16"
	case ValueS32:
		return "s32"
	case ValueU32:
		return "u32"
	case ValueS64:
		return "s64"
	case ValueU64:
		return "u64"
	case ValueDouble:
		return "dbl"
	case ValueShortString:
		return "ss"
	case ValueLongString:
		return "ls"
	default:
		return ""
	}
}

// ParseString parses a string in form type:value into a value and type.
func ParseString(str string) (any, ValueType, error) {
	sepPos := strings.Index(str, ":")
	if sepPos == -1 {
		return nil, ValueUndefined, fmt.Errorf("expression %q does not match pattern type:value", str)
	}

	typStr := str[:sepPos]
	valueStr := str[sepPos+1:]

	typ, err := StringToType(typStr)
	if err != nil {
		return nil, ValueUndefined, err
	}

	var value any
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
	case ValueDouble:
		value, err = strconv.ParseFloat(valueStr, 64)
	case ValueShortString, ValueLongString:
		value = valueStr
	default:
		return nil, ValueUndefined, fmt.Errorf("unsupported type: %s", typStr)
	}

	if err != nil {
		return nil, ValueUndefined, fmt.Errorf("failed to parse value: %s", err)
	}

	return value, typ, nil
}

package surp

// Message type identifiers
type MsgType uint8

const (
	ProtocolVersion1 uint8 = 1
)

const (
	MsgTypeIS  MsgType = 0x01
	MsgTypeSET MsgType = 0x02
	MsgTypeGET MsgType = 0x03
)

// Value types
type ValueType uint8

const (
	ValueUndefined   ValueType = 0x00
	ValueBool        ValueType = 0x01
	ValueU8          ValueType = 0x02
	ValueS8          ValueType = 0x03
	ValueU16         ValueType = 0x04
	ValueS16         ValueType = 0x05
	ValueU32         ValueType = 0x06
	ValueS32         ValueType = 0x07
	ValueU64         ValueType = 0x08
	ValueS64         ValueType = 0x09
	ValueFloat64     ValueType = 0x0A
	ValueShortString ValueType = 0x0B
	ValueLongString  ValueType = 0x0C
)

// Value type names
var VALUE_TYPE_NAMES = map[ValueType]string{
	ValueUndefined:   "undefined",
	ValueBool:        "bool",
	ValueU8:          "u8",
	ValueS8:          "s8",
	ValueU16:         "u16",
	ValueS16:         "s16",
	ValueU32:         "u32",
	ValueS32:         "s32",
	ValueU64:         "u64",
	ValueS64:         "s64",
	ValueFloat64:     "float64",
	ValueShortString: "ss",
	ValueLongString:  "ls",
}

// Metadata keys
type MetadataKey uint8

const (
	MetaRO          MetadataKey = 0x01
	MetaMin         MetadataKey = 0x02
	MetaMax         MetadataKey = 0x03
	MetaUnit        MetadataKey = 0x04
	MetaDescription MetadataKey = 0x05
)

// Metadata key names
var METADATA_KEY_NAMES = map[MetadataKey]string{
	MetaRO:          "ro",
	MetaMin:         "min",
	MetaMax:         "max",
	MetaUnit:        "unit",
	MetaDescription: "description",
}

// Metadata entry structure
type MetadataEntry struct {
	Key       MetadataKey
	ValueType ValueType
	Value     any
}

// Common message header
type MessageHeader struct {
	Version uint8
	MsgType MsgType
}

// MessageIS represents the "Inform State" message
type MessageIS struct {
	TTL       uint16
	Name      string
	ValueType ValueType
	Value     any
	Metadata  []MetadataEntry
}

// MessageSET represents the "Set Value" message
type MessageSET struct {
	Name      string
	ValueType ValueType
	Value     any
}

// MessageGET represents the "Get/Subscribe" message
type MessageGET struct {
	TTL  uint16
	Name string
}

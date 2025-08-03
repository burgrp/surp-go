package surp

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/burgrp/surp-go/pkg/pb"
)

var valueParsers = map[string]func(string) (*pb.Value, error){
	"bool":    parseBool,
	"u8":      parseU64,
	"u16":     parseU64,
	"u32":     parseU64,
	"u64":     parseU64,
	"s8":      parseS64,
	"s16":     parseS64,
	"s32":     parseS64,
	"s64":     parseS64,
	"float64": parseF64,
	"ss":      parseString,
	"ls":      parseString,
}

// ParseString parses a string into a Value based on a type name.
func ParseString(valueStr string, typeName string) (*pb.Value, error) {
	parser, ok := valueParsers[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown value type %q", typeName)
	}
	return parser(valueStr)
}

func parseBool(s string) (*pb.Value, error) {
	v := strings.ToUpper(s) == "TRUE" || s == "1"
	return &pb.Value{Value: &pb.Value_BoolValue{BoolValue: v}}, nil
}

func parseU64(s string) (*pb.Value, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &pb.Value{Value: &pb.Value_U64Value{U64Value: v}}, nil
}

func parseS64(s string) (*pb.Value, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &pb.Value{Value: &pb.Value_S64Value{S64Value: v}}, nil
}

func parseF64(s string) (*pb.Value, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &pb.Value{Value: &pb.Value_F64Value{F64Value: v}}, nil
}

func parseString(s string) (*pb.Value, error) {
	return &pb.Value{Value: &pb.Value_StrValue{StrValue: s}}, nil
}

var metadataKeyNames = map[pb.MetadataKey]string{
	pb.MetadataKey_META_RO:          "ro",
	pb.MetadataKey_META_MIN:         "min",
	pb.MetadataKey_META_MAX:         "max",
	pb.MetadataKey_META_UNIT:        "unit",
	pb.MetadataKey_META_DESCRIPTION: "description",
}

// StringToMetadataKey converts a string to a MetadataKey.
func StringToMetadataKey(s string) (pb.MetadataKey, error) {
	for k, name := range metadataKeyNames {
		if name == s {
			return k, nil
		}
	}
	return 0, fmt.Errorf("unknown metadata key %q", s)
}

// MetadataKeyToString converts a MetadataKey to its string form.
func MetadataKeyToString(key pb.MetadataKey) string {
	return metadataKeyNames[key]
}

// GetMetadataValue finds a metadata entry by key.
func GetMetadataValue(metadata []*pb.MetadataEntry, key pb.MetadataKey) *pb.Value {
	for _, entry := range metadata {
		if entry.GetKey() == key {
			return entry.Value
		}
	}
	return nil
}

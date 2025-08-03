package commands

import (
	"errors"
	"fmt"
	"strings"

	surp "github.com/burgrp/surp-go/pkg"
	pb "github.com/burgrp/surp-go/pkg/pb"
	"github.com/burgrp/surp-go/pkg/provider"
	"github.com/spf13/cobra"
)

func GetProvideCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "provide <name> <type> <value> [meta-key:meta-value ...]",
		Short: "Provide a register",
		Long: `Provides a register with the specified name, value and metadata.
Subsequent values are read from stdin and are written to stdout.
Default type is int, if not specified otherwise in metadata.`,
		RunE: runProvide,
	}

	cmd.Args = cobra.MinimumNArgs(3)

	return cmd
}

func runProvide(cmd *cobra.Command, args []string) error {
	env, err := provider.GetEnvironment()
	if err != nil {
		return err
	}

	regName := args[0]
	regType := args[1]
	regValueStr := args[2]

	regValue, err := surp.ParseString(regValueStr, regType)
	if err != nil {
		return fmt.Errorf("failed to parse value %q: %w", regValueStr, err)
	}

	metadata := make([]*pb.MetadataEntry, len(args)-3)
	for i, arg := range args[3:] {
		kv := strings.SplitN(arg, ":", 2)
		if len(kv) != 2 {
			return errors.New("metadata must be in the form key:value")
		}
		metaKey, err := surp.StringToMetadataKey(kv[0])
		if err != nil {
			return fmt.Errorf("invalid metadata key %q: %w", kv[0], err)
		}

		var metaType string
		switch metaKey {
		case pb.MetadataKey_META_RO:
			metaType = "bool"
		case pb.MetadataKey_META_MIN, pb.MetadataKey_META_MAX:
			metaType = regType
		case pb.MetadataKey_META_UNIT:
			metaType = "ss"
		case pb.MetadataKey_META_DESCRIPTION:
			metaType = "ls"
		}

		metaValue, err := surp.ParseString(kv[1], metaType)
		if err != nil {
			return fmt.Errorf("invalid metadata value %q for key %q: %w", kv[1], kv[0], err)
		}
		metadata[i] = &pb.MetadataEntry{
			Key:   metaKey,
			Value: metaValue,
		}
	}

	roVal := surp.GetMetadataValue(metadata, pb.MetadataKey_META_RO)
	ro := roVal != nil && roVal.GetBoolValue()

	println("registry:", env.Registry)
	println("name:", regName)
	println("type:", regType)
	println("value:", fmt.Sprintf("%v", regValue))
	for _, meta := range metadata {
		println("(", surp.MetadataKeyToString(meta.GetKey()), "=", fmt.Sprintf("%v", meta.Value), ")")
	}
	println("read-only:", ro)
	/*
		group, err := surp.JoinGroup(env.Interface, env.Group, false)
		if err != nil {
			return err
		}

		value, err := parseString(valueStr, typ)
		if err != nil {
			return err
		}

		var pro *provider.Register[any]
		pro = provider.NewAnyRegister(name, value, typ, !ro, metadata, func(value surp.Optional[any]) {
			pro.SyncValue(value)
			fmt.Println(value)
		})

		err = group.AddProviders(pro)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			v := scanner.Text()
			value, err = parseString(v, typ)
			if err != nil {
				println(err.Error())
			}
			pro.SyncValue(value)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	*/
	return nil

}

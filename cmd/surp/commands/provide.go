package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	surp "github.com/burgrp/surp-go/pkg"
	"github.com/burgrp/surp-go/pkg/provider"
	"github.com/spf13/cobra"
)

func GetProvideCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "provide <name> <value> [meta-key:meta-value ...]",
		Short: "Provide a register",
		Long: `Provides a register with the specified name, value and metadata.
Subsequent values are read from stdin and are written to stdout.
Default type is int, if not specified otherwise in metadata.`,
		RunE: runProvide,
	}

	cmd.Flags().BoolP("read-only", "r", false, "Make the register read-only.")
	cmd.Args = cobra.MinimumNArgs(2)

	return cmd
}

func runProvide(cmd *cobra.Command, args []string) error {

	env, err := surp.GetEnvironment()
	if err != nil {
		return err
	}

	name := args[0]
	valueStr := args[1]

	metadata := make(map[string]string, len(args)-2)
	for _, arg := range args[2:] {
		kv := strings.SplitN(arg, ":", 2)
		if len(kv) != 2 {
			return errors.New("metadata must be in the form key:value")
		}
		metadata[kv[0]] = kv[1]
	}

	typ := "int"
	if t, ok := metadata["type"]; ok {
		typ = t
	}

	ro, err := cmd.Flags().GetBool("read-only")
	if err != nil {
		return err
	}

	group, err := surp.JoinGroup(env.Interface, env.Group)
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

	group.AddProviders(pro)

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

	return nil

}

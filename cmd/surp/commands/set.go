package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
	"github.com/burgrp/surp-go/pkg/consumer"
	"github.com/spf13/cobra"
)

func GetSetCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "set <register> <value>",
		Short: "Write a register",
		Long: `Writes the specified register.
With --stay flag, the command will remain connected, read values from stdin and write any changes to stdout.
Values are specified as JSON expressions, e.g. true, false, 3.14, "hello world" or null.`,
		RunE: runSet,
	}

	cmd.Flags().BoolP("stay", "s", false, "Stay connected, read values from stdin and write changes to stdout")
	cmd.Flags().DurationP("timeout", "o", surp.SyncTimeout, "Timeout for waiting for the register to be set")
	cmd.Args = cobra.ExactArgs(2)

	return cmd
}

func parseString(value string, typ string) (surp.Optional[any], error) {

	var undefined surp.Optional[any]

	if value == "null" {
		return undefined, nil
	}

	switch typ {
	case "string":
		return surp.NewDefined(any(value)), nil
	case "int":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return undefined, err
		}
		return surp.NewDefined(any(v)), nil
	case "float":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return undefined, err
		}
		return surp.NewDefined(any(v)), nil
	case "bool":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return undefined, err
		}
		return surp.NewDefined(any(v)), nil
	default:
		return undefined, fmt.Errorf("unsupported type: %s", typ)
	}
}

func setRegisterValue(register *consumer.Register[any], desired string, timeout time.Duration, syncs chan surp.Optional[any]) error {
	to := time.After(timeout)

Wait:
	for {
		select {
		case <-to:
			return errors.New("timeout waiting for register to be set")
		case actual := <-syncs:
			if register.GetMetadata().IsDefined() {
				typ := register.GetMetadata().Get()["type"]
				if typ != "" {
					des, err := parseString(desired, typ)
					if err != nil {
						return err
					}

					if actual == des {
						println(desired)
						break Wait
					}

					register.SetValue(des)
				}
			}
		}
	}

	return nil
}

func runSet(cmd *cobra.Command, args []string) error {

	env, err := surp.GetEnvironment()
	if err != nil {
		return err
	}

	name := args[0]
	stay, err := cmd.Flags().GetBool("stay")
	if err != nil {
		return err
	}

	timeout, error := cmd.Flags().GetDuration("timeout")
	if error != nil {
		return error
	}

	group, err := surp.JoinGroup(env.Interface, env.Group)
	if err != nil {
		return err
	}

	syncs := make(chan surp.Optional[any])

	register := consumer.NewAnyRegister(name, func(value surp.Optional[any]) {
		syncs <- value
	})

	group.AddConsumers(register)

	setRegisterValue(register, args[1], timeout, syncs)

	if stay {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			v := scanner.Text()
			setRegisterValue(register, v, timeout, syncs)
		}
		if err := scanner.Err(); err != nil {
			return err
		}

	}

	return nil
}

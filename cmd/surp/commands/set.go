package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/consumer"
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
	cmd.Flags().DurationP("timeout", "o", 30*time.Second, "Timeout for waiting for the register to be set")
	cmd.Args = cobra.ExactArgs(2)

	return cmd
}

// func jsonEquals(a, b string) bool {
// 	valueA := new(interface{})
// 	err := json.Unmarshal([]byte(a), valueA)
// 	if err != nil {
// 		return false
// 	}

// 	valueB := new(interface{})
// 	err = json.Unmarshal([]byte(b), valueB)
// 	if err != nil {
// 		return false
// 	}

// 	equals, err := reflect.DeepEqual(valueA, valueB), nil
// 	if err != nil {
// 		return false
// 	}

// 	return equals
// }

func parseString(value string, typ string) (any, error) {
	switch typ {
	case "string":
		return any(value), nil
	case "int":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return any(v), nil
	case "float":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return any(v), nil
	case "bool":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return any(v), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}

func setRegisterValue(register *consumer.Register[any], desired string, timeout time.Duration, updates chan surp.Optional[any]) error {
	to := time.After(timeout)

Wait:
	for {
		select {
		case <-to:
			return errors.New("timeout waiting for register to be set")
		case actual := <-updates:
			if register.GetMetadata().IsDefined() {
				typ := register.GetMetadata().Get()["type"]
				if typ != "" {
					v, err := parseString(desired, typ)
					if err != nil {
						return err
					}
					d := surp.NewDefined(v)

					if actual == d {
						break Wait
					}

					register.SetValue(d)
				}
			}
		}
	}

	return nil
}

func runSet(cmd *cobra.Command, args []string) error {

	env, err := GetEnvironment()
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

	updates := make(chan surp.Optional[any])

	register := consumer.NewAnyRegister(name, func(value surp.Optional[any]) {
		updates <- value
	})

	group.AddConsumers(register)

	setRegisterValue(register, args[1], timeout, updates)

	if stay {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			v := scanner.Text()
			setRegisterValue(register, v, timeout, updates)
		}
		if err := scanner.Err(); err != nil {
			return err
		}

	}

	return nil
}

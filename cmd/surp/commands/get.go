package commands

import (
	"fmt"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/consumer"
	"github.com/spf13/cobra"
)

func GetGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <register>",
		Short: "Read a register",
		Long: `Reads the specified register.
	With --stay flag, the command will remain connected and write any changes to stdout.`,
		RunE: runGet,
	}

	cmd.Flags().StringP("type", "t", "", "Type of the register: string, int, float, bool")
	cmd.Flags().BoolP("stay", "s", false, "Stay connected and write changes to stdout")
	cmd.Args = cobra.ExactArgs(1)

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {

	env, err := GetEnvironment()
	if err != nil {
		return err
	}

	name := args[0]
	stay, err := cmd.Flags().GetBool("stay")
	if err != nil {
		return err
	}

	typ, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}

	group, err := surp.JoinGroup(env.Interface, env.Group)
	if err != nil {
		return err
	}

	values := make(chan surp.Optional[any])

	print("Reading register: ", name, "\n")
	group.AddConsumers(
		consumer.NewAnyRegister(name, typ, func(value surp.Optional[any]) {
			values <- value
		}),
	)

	for {
		fmt.Println(<-values)
		if !stay {
			break
		}
	}

	return nil
}

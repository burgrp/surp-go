package main

import (
	"os"

	"github.com/burgrp-go/surp/cmd/surp/commands"
	"github.com/spf13/cobra"
)

func main() {
	RootCmd := &cobra.Command{
		Use:   "surp",
		Short: "surp is a command line tool for working with registers over SURP protocol.",
		Long: `The reg command is a command line tool for working with registers over SURP protocol.
	It allows you to read, write and list registers.
	Furthermore it can provide a 'virtual' register which is convenient for debugging of consumers of the register.

	Two environment variables are required:
	- SURP_IF: The network interface to bind to
	- SURP_GROUP: The SURP group name to join

	For more information on registers over SURP, see: https://github.com/burgrp/surp-go .`,
		SilenceUsage: true,
	}

	RootCmd.AddCommand(
		commands.GetGetCommand(),
		commands.GetSetCommand(),
		commands.GetListCommand(),
		commands.GetVersionCommand(),
	)

	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

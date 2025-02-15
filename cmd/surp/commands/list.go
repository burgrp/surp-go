package commands

import (
	"fmt"
	"strings"
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/spf13/cobra"
)

func GetListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [<reg1> <reg2> ...]",
		Short: "List all known registers",
		Long: `Lists all known registers.
	With --stay flag, the command will remain connected and write any changes to stdout.
	If registers are specified, only those will be listed.`,
		RunE: runList,
	}

	cmd.Flags().BoolP("stay", "s", false, "Stay connected infinitely and write changes to stdout")
	cmd.Flags().DurationP("timeout", "t", surp.SyncTimeout, "Timeout for waiting for the registers")
	cmd.Flags().BoolP("values", "v", false, "Do not print values")
	cmd.Flags().BoolP("meta", "m", false, "Do not print metadata")

	return cmd
}

func passNameFilter(name string, args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if strings.Contains(name, arg) {
			return true
		}
	}
	return false
}

func runList(cmd *cobra.Command, args []string) error {

	env, err := GetEnvironment()
	if err != nil {
		return err
	}

	stay, err := cmd.Flags().GetBool("stay")
	if err != nil {
		return err
	}

	timeout, error := cmd.Flags().GetDuration("timeout")
	if error != nil {
		return error
	}

	noValues, err := cmd.Flags().GetBool("values")
	if err != nil {
		return err
	}

	noMeta, err := cmd.Flags().GetBool("meta")
	if err != nil {
		return err
	}

	group, err := surp.JoinGroup(env.Interface, env.Group)
	if err != nil {
		return err
	}

	allSynced := make(map[string]struct{})

	group.OnSync(func(message *surp.Message) {
		name := message.Name
		_, synced := allSynced[name]
		if passNameFilter(name, args) && (!synced || stay) {
			typ := message.Metadata["type"]
			var value surp.Optional[any]
			if message.Value.IsDefined() {
				v, ok := surp.DecodeGeneric(message.Value.Get(), typ)
				if !ok {
					return
				}
				value = surp.NewDefined[any](v)
			}
			valueStr := ""
			if !noValues {
				valueStr = fmt.Sprintf("=%s", value.String())
			}
			metaStr := ""
			if !noMeta {
				for k, v := range message.Metadata {
					if metaStr != "" {
						metaStr += " "
					}
					metaStr += fmt.Sprintf("%s:%s", k, v)
				}
				metaStr = " \t[" + metaStr + "]"
			}
			fmt.Printf("%s%s%s\n", name, valueStr, metaStr)
			allSynced[name] = struct{}{}
		}
	})

	var to <-chan time.Time
	if !stay {
		to = time.After(timeout)
	}

	select {
	case <-to:
	case <-cmd.Context().Done():
	}

	return nil
}

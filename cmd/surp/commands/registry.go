package commands

import (
	"log/slog"

	surp "github.com/burgrp/surp-go/pkg"
	reg "github.com/burgrp/surp-go/pkg/registry"
	"github.com/spf13/cobra"
)

const defaultRegistryAddress = ":5530"

func GetRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Starts SURP registry service",
		Long:  `Starts SURP registry service. For now we only support SURP over UDP. In future versions we will add MQTT, REST, Prometheus and other protocols.`,
		RunE:  runRegistry,
	}

	cmd.Flags().StringP("address", "a", defaultRegistryAddress, "Listen address for SURP registry")

	return cmd
}

func runRegistry(cmd *cobra.Command, args []string) error {

	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return err
	}

	logger := slog.Default()

	socket, err := surp.NewSocket(address, logger)
	if err != nil {
		return err
	}

	registry := reg.NewRegistry(logger)
	native := reg.NewNative(socket, registry, logger)

	logger.Info("SURP registry started", "address", address)

	surp.RunWorkers(socket, registry, native)

	return nil
}

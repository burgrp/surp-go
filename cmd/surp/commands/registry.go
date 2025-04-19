package commands

import (
	"fmt"
	"log/slog"
	"os"

	surp "github.com/burgrp/surp-go/pkg"
	"github.com/burgrp/surp-go/pkg/cli"
	reg "github.com/burgrp/surp-go/pkg/registry"
	"github.com/phsym/console-slog"
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

func NewLogger(cmd *cobra.Command) (*slog.Logger, error) {

	logLevel, err := cmd.Flags().GetString("log")
	if err != nil {
		return nil, err
	}

	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", logLevel)
	}

	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{Level: level}),
	)
	return logger, nil
}

func runRegistry(cmd *cobra.Command, args []string) error {

	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return err
	}

	logger, err := NewLogger(cmd)
	if err != nil {
		return err
	}

	socket, err := surp.NewSocket(address, logger)
	if err != nil {
		return err
	}

	registry := reg.NewRegistry(logger)
	native := reg.NewNative(socket, registry, logger)

	logger.Info("SURP registry started", "address", address)
	cli.RunWorkers(socket, registry, native)
	logger.Info("SURP registry stopped")

	return nil
}

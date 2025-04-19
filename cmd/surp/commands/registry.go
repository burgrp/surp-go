package commands

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	surp "github.com/burgrp/surp-go/pkg"
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

	registry := surp.NewRegistry(logger)

	native := surp.NewNative(socket, registry, logger)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	socket.Start(ctx, &wg)
	registry.Start(ctx, &wg)
	native.Start(ctx, &wg)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	logger.Info("Shutting down...")
	cancel()
	wg.Wait()

	return nil
}

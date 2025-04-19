package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
)

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

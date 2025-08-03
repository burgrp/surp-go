package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type Worker interface {
	Run(ctx context.Context) error
}

func RunWorkers(workers ...Worker) error {
	ctx, cancel := context.WithCancel(context.Background())
	var g errgroup.Group

	for _, worker := range workers {
		g.Go(func() error {
			err := worker.Run(ctx)
			if err != nil {
				cancel()
			}
			return err
		})
	}

	g.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return nil
		case <-sigs:
			cancel()
			return nil
		}
	})

	return g.Wait()
}

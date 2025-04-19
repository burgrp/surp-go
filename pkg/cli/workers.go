package cli

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Worker interface {
	Run(ctx context.Context)
}

func RunWorkers(workers ...Worker) {

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	for _, worker := range workers {
		wg.Add(1)
		go func() {
			worker.Run(ctx)
			wg.Done()
		}()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()
	wg.Wait()
}

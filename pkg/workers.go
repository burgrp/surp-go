package surp

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Worker interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
}

func RunWorkers(workers ...Worker) {

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	for _, worker := range workers {
		worker.Start(ctx, &wg)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()
	wg.Wait()

}

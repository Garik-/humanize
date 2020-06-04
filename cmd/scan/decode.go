package main

import (
	"context"
	"github.com/Garik-/humanize/pkg/midi"
	"go.uber.org/zap"
	"os"
	"sync"
)

type result struct {
	name   string
	events []*midi.Event
	err    error
}

func decodeFile(name string) *result {
	out := &result{name: name}
	f, err := os.Open(name)
	if err != nil {
		out.err = err
		return out
	}

	defer f.Close()

	decoder := midi.NewDecoder(f)
	err = decoder.Decode()
	if err != nil {
		out.err = err
		return out
	}

	out.events = decoder.Events
	return out
}

func decodeRoutine(ctx context.Context, path string, goroutines <-chan struct{}, out chan<- *result, wg *sync.WaitGroup) {
	log := decoderLog.Named("decodeRoutine")
	defer wg.Done()

	select {
	case out <- decodeFile(path):
	case <-ctx.Done():
		log.Debug("context done", zap.String("path", path))
	}
	<-goroutines
}

func decodeWorker(ctx context.Context, paths <-chan string, cntRoutines int) (<-chan *result, <-chan struct{}) {
	log := decoderLog.Named("decodeWorker")
	out := make(chan *result)
	done := make(chan struct{}, 1)

	go func() {
		var wg sync.WaitGroup
		goroutines := make(chan struct{}, cntRoutines)

	loop:
		for path := range paths {
			select {
			case goroutines <- struct{}{}:
			case <-ctx.Done():
				log.Debug("context done")
				break loop
			}
			wg.Add(1)
			go decodeRoutine(ctx, path, goroutines, out, &wg)
		}

		wg.Wait()
		close(goroutines)
		close(out)

		done <- struct{}{}
		close(done)
	}()

	return out, done
}

package main

import (
	"context"
	"github.com/Garik-/humanize/pkg/midi"
	"log"
	"os"
	"sync"
)

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

func decodeWorker(ctx context.Context, paths <-chan string, cntRoutines int) (<-chan *result, <-chan struct{}) {
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
				log.Println("decodeWorker context done")
				break loop
			}
			wg.Add(1)
			go func(ctx context.Context, path string, goroutines <-chan struct{}, out chan<- *result, wg *sync.WaitGroup) {
				defer wg.Done()

				select {
				case out <- decodeFile(path):
				case <-ctx.Done():
					log.Printf("decodeFile %s context done\n", path)
				}
				<-goroutines

			}(ctx, path, goroutines, out, &wg)
		}

		wg.Wait()
		close(goroutines)
		close(out)

		done <- struct{}{}
		close(done)
	}()

	return out, done
}

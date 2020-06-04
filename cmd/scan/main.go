package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Garik-/humanize/pkg/midi"
	"log"
	"os"
	"sync"
)

const (
	maxGoroutines = 10
)

var (
	listFlag = flag.String("l", "", "The path to the list of midi files,\nfind . -type f -name \"*.mid\" > midi_list.txt")
	maxFlag  = flag.Int("p", maxGoroutines, "Number of files processed in parallel, must be > 0")
)

type result struct {
	name   string
	events []*midi.Event
	err    error
}

// note -> type -> velocity
type velocityMap map[uint8]map[uint8]map[uint8]bool

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

func readList(file *os.File) <-chan string {
	out := make(chan string)

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	go func() {
		for scanner.Scan() {
			out <- scanner.Text()
		}
		close(out)
	}()

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

func newVelocityMap(parent context.Context, paths <-chan string, cntRoutines int) (velocityMap, error) {
	ctx, cancel := context.WithCancel(parent)
	results, done := decodeWorker(ctx, paths, cntRoutines)

	defer func() {
		log.Println("newVelocityMap cancel")
		cancel()
		<-done // wait decodeWorker closed
	}()

	m := make(velocityMap)
	i := 0

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}

		log.Printf("name: %s, events: %d", result.name, len(result.events))

		i++
		if i == 10 {
			break
		}
	}

	return m, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *listFlag == "" {
		flag.Usage()
		return
	}

	if *maxFlag <= 0 {
		flag.Usage()
		return
	}

	f, err := os.Open(*listFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// done := make(chan os.Signal, 1)
	// signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	paths := readList(f)
	var m velocityMap
	m, err = newVelocityMap(context.Background(), paths, *maxFlag)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%v", m)
}

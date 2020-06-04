package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Garik-/humanize/pkg/midi"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{}, 1)

	defer func() {
		f.Close()

		done <- struct{}{}
		close(done)
	}()

	go func() {
		doneSignal := make(chan os.Signal, 1)
		signal.Notify(doneSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-doneSignal:
		case <-done:
		}

		cancel()
	}()

	paths := readList(f)
	var m velocityMap
	m, err = newVelocityMap(ctx, paths, *maxFlag)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%v", m)
}

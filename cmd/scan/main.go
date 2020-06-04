package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"go.uber.org/zap"
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

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		enableDebugLogging(logger)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *listFlag == "" || *maxFlag <= 0 {
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

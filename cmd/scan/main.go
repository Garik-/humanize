package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	maxGoroutines = 2
)

var (
	listFlag = flag.String("l", "", "The path to the list of midi files,\nfind . -type f -name \"*.mid\" > midi_list.txt")
	outFlag  = flag.String("o", "", "The path to output json file")
	maxFlag  = flag.Int("p", maxGoroutines, "Number of files processed in parallel, must be > 0")
)

func readList(file *os.File) (<-chan string, error) {
	out := make(chan string)

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	go func() {
		for scanner.Scan() {
			out <- scanner.Text()
		}
		close(out)
	}()

	return out, nil
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

	if *listFlag == "" || *outFlag == "" || *maxFlag <= 0 {
		flag.Usage()
		return
	}

	in, err := os.Open(*listFlag)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.OpenFile(*outFlag, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		in.Close()
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{}, 1)

	defer func() {
		out.Close()
		in.Close()

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

	paths, err := readList(in)
	if err != nil {
		log.Fatal(err)
	}

	var m velocityMap
	m, err = newVelocityMap(ctx, paths, *maxFlag)
	if err != nil {
		log.Fatal(err)
	}

	// note > type > []velocity
	data := make(map[uint8]map[uint8][]int)
	for note, types := range m {
		for msgType, velocity := range types {
			mainLog.Debug("map", zap.Uint8("note", note), zap.Uint8("msgType", msgType), zap.Int("velocity", len(velocity)))

			i := 0
			keys := make([]int, len(velocity))
			for k := range velocity {
				keys[i] = int(k)
				i++
			}

			if _, ok := data[note]; ok {
				data[note][msgType] = keys
			} else {
				msgTypes := make(map[uint8][]int)
				msgTypes[msgType] = keys
				data[note] = msgTypes
			}
		}
	}

	encoder := json.NewEncoder(out)
	err = encoder.Encode(data)
	if err != nil {
		log.Fatal(err)
	}
}

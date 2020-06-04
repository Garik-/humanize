package main

import (
	"context"
	"go.uber.org/zap"
)

// note -> type -> velocity
type velocityMap map[uint8]map[uint8]map[uint8]bool

func newVelocityMap(parent context.Context, paths <-chan string, cntRoutines int) (velocityMap, error) {
	log := velocityMapLog.Named("newVelocityMap")
	ctx, cancel := context.WithCancel(parent)
	results, done := decodeWorker(ctx, paths, cntRoutines)

	defer func() {
		log.Debug("cancel")
		cancel()
		<-done // wait decodeWorker closed
	}()

	m := make(velocityMap)
	i := 0

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}

		log.Debug("result", zap.String("name", result.name), zap.Int("events", len(result.events)))

		i++
		if i == 10 {
			break
		}
	}

	return m, nil
}

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

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}

		log.Debug("result", zap.String("name", result.name), zap.Int("events", len(result.events)))

		for _, event := range result.events {
			if event.Velocity == 0 {
				continue
			}

			if _, ok := m[event.Note]; ok {
				m[event.Note][event.MsgType][event.Velocity] = true
			} else {
				velocity := make(map[uint8]bool)
				velocity[event.Velocity] = true

				msgType := make(map[uint8]map[uint8]bool)
				msgType[event.MsgType] = velocity

				m[event.Note] = msgType
			}
		}
	}

	return m, nil
}

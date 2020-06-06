package main

import (
	"context"
	"go.uber.org/zap"
)

type velocityMap map[uint8]bool
type positionMap map[int]velocityMap
type typeMap map[uint8]positionMap

// note -> type -> position -> velocity
type noteMap map[uint8]typeMap

func newVelocityMap(parent context.Context, paths <-chan string, cntRoutines int) (noteMap, error) {
	log := velocityMapLog.Named("newVelocityMap")
	ctx, cancel := context.WithCancel(parent)
	results, done := decodeWorker(ctx, paths, cntRoutines)

	defer func() {
		log.Debug("cancel")
		cancel()
		<-done // wait decodeWorker closed
	}()

	m := make(noteMap)

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}

		log.Debug("result", zap.String("name", result.name), zap.Int("tracks", len(result.tracks)))

		for _, track := range result.tracks {

			for _, event := range track.Events {
				if event.Velocity == 0 {
					continue
				}

				log.Debug("event", zap.Uint8("note", event.Note), zap.Int("position", event.QuarterPosition))

				if _, ok := m[event.Note]; ok { // check type
					if _, ok := m[event.Note][event.MsgType]; ok { // check position
						if _, ok := m[event.Note][event.MsgType][event.QuarterPosition]; ok { // check Velocity
							m[event.Note][event.MsgType][event.QuarterPosition][event.Velocity] = true
						} else {
							velocity := make(velocityMap)
							velocity[event.Velocity] = true

							m[event.Note][event.MsgType][event.QuarterPosition] = velocity
						}
					} else {
						velocity := make(velocityMap)
						velocity[event.Velocity] = true

						position := make(positionMap)
						position[event.QuarterPosition] = velocity

						m[event.Note][event.MsgType] = position
					}
				} else {
					velocity := make(velocityMap)
					velocity[event.Velocity] = true

					position := make(positionMap)
					position[event.QuarterPosition] = velocity

					msgType := make(typeMap)
					msgType[event.MsgType] = position

					m[event.Note] = msgType
				}
			}
		}
	}

	return m, nil
}

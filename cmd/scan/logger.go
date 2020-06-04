package main

import "go.uber.org/zap"

var decoderLog = zap.NewNop()
var velocityMapLog = zap.NewNop()

func enableDebugLogging(l *zap.Logger) {
	decoderLog = l
	velocityMapLog = l
}

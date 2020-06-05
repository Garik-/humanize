package main

import "go.uber.org/zap"

var decoderLog = zap.NewNop()
var velocityMapLog = zap.NewNop()
var mainLog = zap.NewNop()

func enableDebugLogging(l *zap.Logger) {
	decoderLog = l
	velocityMapLog = l
	mainLog = l
}

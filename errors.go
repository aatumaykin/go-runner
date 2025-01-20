package go_runner

import "errors"

var (
	ErrInterruptedBySignal = errors.New("process interrupted by signal")
)

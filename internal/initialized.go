package internal

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	inits = make([]func(), 0)
	exits = make([]func(), 0)
)

func AddInitialized(apply func()) { inits = append(inits, apply) }
func AddExited(apply func())      { exits = append(exits, apply) }
func Initialized() {
	for _, yield := range inits {
		yield()
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func(ch chan os.Signal) {
		<-ch
		for _, yield := range exits {
			yield()
		}
		os.Exit(0)
	}(osSignal)
}

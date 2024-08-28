package signalx

import (
	"context"
	"os"
	"os/signal"
)

// SignalCtx will set up a context that will be cancelled if any of the given signals are received.
func SignalCtx(parent context.Context, signals ...os.Signal) context.Context {
	if len(signals) == 0 {
		panic("no signals passed to SignalContext")
	}
	ctx, cancel := context.WithCancel(parent)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, signals...)
	go func() {
		defer cancel()
		<-sigs
	}()
	return ctx
}

// SignalExitCtx will set up a context that will be cancelled if any of the given signals are received.
// If a second signal is received, then [os.Exit] will be called with a non-zero exit code.
func SignalExitCtx(parent context.Context, signals ...os.Signal) context.Context {
	if len(signals) == 0 {
		panic("no signals passed to SignalContext")
	}
	ctx, cancel := context.WithCancel(parent)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, signals...)
	go func() {
		defer cancel()
		<-sigs
		cancel()
		<-sigs
		os.Exit(1)
	}()
	return ctx
}

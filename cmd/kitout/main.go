package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vwall/kitout/internal/cli"
)

func main() {
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(stopSignals)
	}
	go func() {
		<-ctx.Done()
		stop()
	}()

	code := cli.RunContext(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

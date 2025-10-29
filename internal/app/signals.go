package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SetupSignalHandler creates a context that is canceled on SIGINT or SIGTERM
func SetupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\n\nReceived signal %s, shutting down gracefully...\n", sig)
		cancel()

		// Force exit after 5 seconds if graceful shutdown fails
		time.Sleep(5 * time.Second)
		fmt.Fprintf(os.Stderr, "Forced shutdown after timeout\n")
		os.Exit(1)
	}()

	return ctx, cancel
}

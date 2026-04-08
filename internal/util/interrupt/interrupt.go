// internal/util/interrupt/interrupt.go

package interrupt

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/llm-d-incubation/batch-gateway/internal/util/logging"
	"k8s.io/klog/v2"
)

// ContextWithSignal monitors OS signals and returns a context that is cancelled on the first
// interrupt-like signal (SIGINT, SIGTERM, os.Interrupt). If grace is greater than zero,
// cancellation is delayed by that duration so in-flight work can finish while the context
// remains valid. A grace of zero cancels the context immediately on the first signal.
// A second signal forces process exit without waiting for the grace timer.
func ContextWithSignal(parent context.Context, grace time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	logger := logr.FromContextOrDiscard(ctx)

	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		if grace > 0 {
			logger.Info("Received shutdown signal, starting graceful shutdown when grace period expires", "signal", sig, "grace", grace)
			time.AfterFunc(grace, cancel)
		} else {
			logger.V(logging.INFO).Info("Received shutdown signal, starting graceful shutdown...", "signal", sig)
			cancel()
		}

		sig = <-signalChan
		logger.V(logging.INFO).Info("Received second shutdown signal, forcing shutdown...", "signal", sig)
		klog.Flush()
		os.Exit(1)
	}()

	return ctx, cancel
}

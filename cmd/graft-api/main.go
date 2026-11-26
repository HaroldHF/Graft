// Command graft-api runs the Graft Control Plane HTTP API and background reaper.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/HaroldHF/Graft/internal/buildinfo"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		slog.Error("graft-api exited with error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if ctx == nil {
		return errors.New("nil context")
	}

	fs := flag.NewFlagSet("graft-api", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print build version and exit")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if *showVersion {
		if _, err := fmt.Fprintln(stdout, buildinfo.Summary("graft-api")); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled before startup: %w", err)
	}

	_, err := fmt.Fprintln(stdout, "graft-api bootstrap ready (Phase 1 HTTP server pending)")
	if err != nil {
		return fmt.Errorf("write startup message: %w", err)
	}
	return nil
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/RokiLai/agent_sync_tool/internal/app"
)

func main() {
	executable, _ := os.Executable()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Main(ctx, os.Args[1:], app.Dependencies{
		Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Executable: executable,
		IsTerminal:       func() bool { info, err := os.Stdin.Stat(); return err == nil && info.Mode()&os.ModeCharDevice != 0 },
		IsOutputTerminal: func() bool { info, err := os.Stdout.Stat(); return err == nil && info.Mode()&os.ModeCharDevice != 0 },
	}))
}

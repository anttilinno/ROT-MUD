package main

import (
	"log/slog"
	"os"

	"rotmud/pkg/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	srv := server.New(logger)
	if err := srv.Start(4000); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"log/slog"
	"os"

	"rotmud/pkg/server"
)

func main() {
	debug := flag.Bool("debug", false, "log player commands and output")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	srv := server.New(logger)
	if err := srv.Start(4000); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

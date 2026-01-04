package main

import (
	"log/slog"
	"os"

	"rotmud/pkg/loader"
	"rotmud/pkg/server"
	"rotmud/pkg/types"
)

func main() {
	// Load configuration
	configPath := "data/config.toml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := loader.LoadConfigFromFile(configPath)
	if err != nil {
		// Fall back to defaults if config not found
		cfg = &loader.Config{
			Server: loader.ServerConfig{
				TelnetPort:    4000,
				WebsocketPort: 4001,
				PulseMs:       250,
				DataPath:      "data",
			},
			Logging: loader.LoggingConfig{
				Level:  "info",
				Format: "text",
			},
			Security: loader.SecurityConfig{
				APIKey: "changeme",
			},
		}
	}

	// Set up structured logging
	logLevel := slog.LevelInfo
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	// Override from environment if set
	if os.Getenv("DEBUG") != "" {
		logLevel = slog.LevelDebug
	}

	// Use JSON logging in production, text for development
	var handler slog.Handler
	logFormat := cfg.Logging.Format
	if envFormat := os.Getenv("LOG_FORMAT"); envFormat != "" {
		logFormat = envFormat
	}

	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}
	logger := slog.New(handler)

	// Create the main server
	srv := server.New(logger)
	srv.DataPath = cfg.Server.DataPath
	srv.SetAPIKey(cfg.Security.APIKey)

	// Start HTTP/WebSocket server in background
	httpServer := server.NewHTTPServer(srv, logger)

	// Wire up metrics callbacks
	srv.Dispatcher.Combat.OnDamage = func(damage int) {
		httpServer.Metrics.RecordDamage(damage)
	}
	srv.Dispatcher.Magic.OnSpellCast = func() {
		httpServer.Metrics.IncrementSpellsCast()
	}

	// Wire up weather control for control weather spell
	srv.Dispatcher.Magic.WeatherControl = func(change int) {
		if srv.GameLoop != nil && srv.GameLoop.WorldTime != nil {
			srv.GameLoop.WorldTime.ControlWeather(change)
		}
	}

	// Wire up quest system kill trigger
	srv.Dispatcher.Combat.OnKill = func(killer, victim *types.Character) {
		if killer == nil || killer.PCData == nil || victim == nil {
			return
		}
		if srv.Quests.OnMobKill(killer, victim) {
			// Notify the player if a quest was updated
			if srv.Dispatcher.Output != nil {
				srv.Dispatcher.Output(killer, "{YQuest progress updated!{x\r\n")
			}
		}
	}

	// Set up metrics updater (runs on pulse)
	srv.SetMetricsUpdater(func(playerCount, npcCount, connCount int, _ int64) {
		httpServer.Metrics.UpdateCounts(playerCount, npcCount)
		httpServer.Metrics.SetConnections(connCount)
	})

	// Wire up command counting for metrics
	srv.SetOnCommand(func() {
		httpServer.Metrics.IncrementCommands()
	})

	go func() {
		httpPort := cfg.Server.WebsocketPort
		logger.Info("starting HTTP/WebSocket server", "port", httpPort)
		if err := httpServer.Start(httpPort); err != nil {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	// Start telnet server (blocking)
	telnetPort := cfg.Server.TelnetPort
	logger.Info("starting ROT MUD server",
		"telnet_port", telnetPort,
		"http_port", cfg.Server.WebsocketPort,
		"config", configPath)

	if err := srv.Start(telnetPort); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

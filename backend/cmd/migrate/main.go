package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lingchou/lingchoubot/backend/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	cfg := config.Load()

	m, err := migrate.New("file://migrations", cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to create migrate instance", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logger.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migration up completed")
	case "down":
		if err := m.Steps(-1); err != nil {
			logger.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migration down completed (1 step)")
	default:
		logger.Error("unknown direction", "direction", *direction)
		os.Exit(1)
	}
}

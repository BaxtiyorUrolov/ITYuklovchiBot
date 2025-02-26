package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v4/stdlib" // To'g'ri `pgx` driver
	"yuklovchiBot/config"
	"yuklovchiBot/pkg/logger"
)

func New(ctx context.Context, cfg config.Config, log logger.Logger) (*sql.DB, error) {
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresDB,
	)

	db, err := sql.Open("pgx", url)
	if err != nil {
		log.Error("error while connecting to db", logger.Error(err))
		return nil, err
	}

	// Maksimal ulanishlarni sozlash
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)

	// Bog'lanishni tekshirish
	if err := db.PingContext(ctx); err != nil {
		log.Error("error while pinging db", logger.Error(err))
		return nil, err
	}

	return db, nil
}

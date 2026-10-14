package postrgesql

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	db *gorm.DB
}

func New(host string, port int, dbname string, user string, password string) (*Storage, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}

	return s.db.WithContext(ctx)
}

func (s *Storage) Ping(ctx context.Context) error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func (s *Storage) Name() string {
	return "postgres"
}

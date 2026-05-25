package main

import (
	"activeTver/internal/pkg/logs"
	"errors"
	"flag"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const op = "cmd.migrator"

func main() {
	logs.Setup()
	defer logs.Flush()

	var storageURL, action, migrationsPath, migrationsTable string

	flag.StringVar(&storageURL, "storage-URL", "", "")
	flag.StringVar(&action, "action", "", "up/down/revert")
	flag.StringVar(&migrationsPath, "migrations-path", "", "")
	flag.StringVar(&migrationsTable, "migrations-table", "", "")
	flag.Parse()

	migration, err := migrate.New("file://"+migrationsPath, fmt.Sprintf("postgres://%s?x-migrations-table=%s&sslmode=disable", storageURL, migrationsTable))
	if err != nil {
		logs.Error(
			err.Error(),
			"migrator",
			op,
		)

		return
	}

	switch action {
	case "up":
		if err = migration.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				logs.Info(
					"no new migrations",
					"migrator",
					op,
				)
			} else {
				logs.Error(
					err.Error(),
					"migrator",
					op,
				)
				return
			}
		}
	case "down":
		if err := migration.Down(); err != nil {
			logs.Error(
				err.Error(),
				"migrator",
				op,
			)

			return
		}
	}

	logs.Info(
		"migrations applied",
		"migrator",
		op,
	)
}

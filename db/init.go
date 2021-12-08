package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	_db         *sql.DB
	_initDBOnce sync.Once
)

func InitDB(path string) error {
	var err error
	_initDBOnce.Do(func() {
		log.Debugf("database path: %s", path)

		parentDir := filepath.Dir(path)
		err = os.MkdirAll(parentDir, 0o755)
		if err != nil {
			err = errors.Wrapf(err, "failed to create parent directory %#v for database", parentDir)
			return
		}

		err = runMigrations(path)
		if err != nil {
			err = errors.Wrapf(err, "error occurred during schema migrations")
			return
		}

		_db, err = sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=10")
		if err != nil {
			err = errors.Wrapf(err, "failed to open SQLite3 database %#v", path)
			return
		}
		err = nil
	})
	return err
}

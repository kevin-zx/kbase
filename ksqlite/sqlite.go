package ksqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// create db connection
func CreateDB(dbfile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return nil, err
	}
	return db, nil
}

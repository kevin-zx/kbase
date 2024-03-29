// 使用 sqlite 作为本地缓存

package kcache

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteCache struct {
	db        *sql.DB
	prefix    string
	tableName string
}

func NewSqliteCache(dbfile string, prefix string) (*sqliteCache, error) {
	conectFmt := "file:%s?cache=shared&mode=rwc"
	db, err := sql.Open("sqlite3", fmt.Sprintf(conectFmt, dbfile))
	if err != nil {
		return nil, err
	}
	tableName := prefix + "_cache"

	sc := &sqliteCache{
		db:        db,
		tableName: tableName,
	}
	err = sc.createCacheTable()
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (c *sqliteCache) createCacheTable() error {
	_, err := c.db.Exec(
		"CREATE TABLE IF NOT EXISTS " + c.tableName + " (key TEXT PRIMARY KEY, value BLOB, created_time DATETIME)",
	)
	if err != nil {
		return err
	}

	return err
}

func (c *sqliteCache) Get(key string) ([]byte, error) {
	key = md5String(key)
	var value []byte
	err := c.db.QueryRow(
		"SELECT value FROM "+
			c.tableName+
			" WHERE key = ?", key).Scan(&value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *sqliteCache) Save(key string, value []byte) error {
	key = md5String(key)
	err := s.Delete(key)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT OR REPLACE INTO "+
			s.tableName+
			" (key, value, created_time) VALUES (?, ?, datetime('now'))",
		key, value)
	return err
}

func (s *sqliteCache) Delete(key string) error {
	key = md5String(key)
	_, err := s.db.Exec(
		"DELETE FROM "+
			s.tableName+
			" WHERE key = ?",
		key)
	return err
}

func (s *sqliteCache) Close() error {
	return s.db.Close()
}

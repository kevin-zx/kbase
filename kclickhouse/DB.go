package kclickhouse

import (
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/jmoiron/sqlx"
)

func CreateClickhouseDB(host string, port int, username string, password string, databese string) (*sqlx.DB, error) {
	xhsdb, err := sqlx.Open("clickhouse", fmt.Sprintf("tcp://%s:%d?username=%s&password=%s&database=%s&read_timeout=1000&write_timeout=2000", host, port, username, password, databese))
	if err != nil {
		return nil, err
	}
	err = xhsdb.Ping()
	if err == nil {
		return xhsdb, nil
	}
	return xhsdb, err
}

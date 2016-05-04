// Common Database Functions
package db

import (
	"database/sql"
	"path/filepath"

  "github.com/nelsam/gorq"
	_ "github.com/mattn/go-sqlite3"
)

func InitDb() error {
	dbPath := filepath.Join(tc.Directories["user_dir"], tc.Directories["data_dir"], "gumshoe.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		PrintDebugf("sql.Open failed for %s", dbPath)
		return err
	}
	gDb = &gorq.DbMap{Db: db, Dialect: gorq.SqliteDialect{}}

	err = initTable(gDb, Show{}, "show")
	if err != nil {
		PrintDebugf("Table show failed to init: %s\n", err)
	}

	err = initTable(gDb, Episode{}, "episode")
	if err != nil {
		PrintDebugf("Table episode failed to init: %s\n", err)
	}

	return err
}

func initTable(dbmap *gorq.DbMap, i interface{}, tableName string) error {
	dbmap.AddTableWithName(i, tableName).SetKeys(true, "ID")
	return dbmap.CreateTablesIfNotExists()
}

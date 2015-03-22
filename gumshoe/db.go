package gumshoe

import (
	"database/sql"
	"path/filepath"

	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
)

func initDb(baseDir string, dbName string) *gorp.DbMap {
	dbPath := filepath.Join(baseDir, dbName+".db")
	db, err := sql.Open("sqlite3", dbPath)
	checkErr(err, "sql.Open failed for "+dbPath)
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	return dbmap
}

func initTable(dbmap *gorp.DbMap, i interface{}, tableName string) {
	dbmap.AddTableWithName(i, tableName).SetKeys(true, "ID")
	err := dbmap.CreateTablesIfNotExists()
	checkErr(err, "Create table failed")
}

package db

import (
	"database/sql"
  "log"

  "github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

var (
	gDb *gorp.DbMap
  checkDBLock = make(chan int)
)

func InitDb(db_file string) error {
	dbPath := db_file
	dbRoot, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("sql.Open failed for %s\n", dbPath)
		return err
	}
	gDb = &gorp.DbMap{Db: dbRoot, Dialect: gorp.SqliteDialect{}}

	err = initTable(gDb, Show{}, "show")
	if err != nil {
		log.Printf("Table Show failed to init: %s\n", err)
    return err
	}

	err = initTable(gDb, Episode{}, "episode")
	if err != nil {
		log.Printf("Table Episode failed to init: %s\n", err)
    return err
	}
	return nil
}

func initTable(dbmap *gorp.DbMap, i interface{}, tableName string) error {
	dbmap.AddTableWithName(i, tableName).SetKeys(true, "ID")
	return dbmap.CreateTablesIfNotExists()
}

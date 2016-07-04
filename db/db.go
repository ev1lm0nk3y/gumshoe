package db

import (
	"database/sql"
	"expvar"
	"log"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

var (
	gDb         *gorp.DbMap
	checkDBLock = make(chan int)
	dbOps       = expvar.NewMap("num_db_ops")
	dbOpened    = expvar.NewMap("db_opened_timestamp")
)

func InitDb(db_file string) error {
	dbRoot, err := sql.Open("sqlite3", db_file)
	if err != nil {
		log.Printf("sql.Open failed for %s\n", db_file)
		return err
	}
	gDb = &gorp.DbMap{Db: dbRoot, Dialect: gorp.SqliteDialect{}}

	err = initTable(gDb, Show{}, "show")
	if err != nil {
		log.Printf("Table Show failed to init: %s\n", err)
		return err
	}
	dbOpened.Add("show", time.Now().Unix())
	dbOps.Add("show", 1)

	err = initTable(gDb, Episode{}, "episode")
	if err != nil {
		log.Printf("Table Episode failed to init: %s\n", err)
		return err
	}
	dbOpened.Add("episode", time.Now().Unix())
	dbOps.Add("episode", 1)

	return nil
}

func initTable(dbmap *gorp.DbMap, i interface{}, tableName string) error {
	dbmap.AddTableWithName(i, tableName).SetKeys(true, "ID")
	return dbmap.CreateTablesIfNotExists()
}

func AddDBOp(table string) {
	dbOps.Add(table, 1)
}

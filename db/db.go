package db

import (
	"log"
	"server/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	Instance        *gorm.DB
	TimestampFunc   = ""
	CreatedDateFunc = ""
)

func Init() {
	var db *gorm.DB
	var err error
	if config.MYSQL_DSN != "" {
		// MySQL setup
		db, err = gorm.Open(mysql.Open(config.MYSQL_DSN), &gorm.Config{
			PrepareStmt: true,
		})
		if err != nil || db == nil {
			log.Fatalf("MySQL DB error: %v", err)
		}
		TimestampFunc = "unix_timestamp()"
		CreatedDateFunc = "date(from_unixtime(created_at))"
	} else if config.SQLITE_FILE != "" {
		// Sqlite setup
		db, err = gorm.Open(sqlite.Open(config.SQLITE_FILE+"?_foreign_keys=on"), &gorm.Config{})
		if err != nil || db == nil {
			log.Fatalf("SQLite DB error: %v", err)
		}
		TimestampFunc = "strftime('%s', 'now')"
		CreatedDateFunc = "date(created_at, 'unixepoch')"
	} else {
		log.Fatal("No database configuration found")
	}
	Instance = db
}

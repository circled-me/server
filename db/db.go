package db

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func Init(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// SkipDefaultTransaction: true,
		PrepareStmt: true,
	})
	if err != nil || db == nil {
		log.Fatalf("DB error: %v", err)
	}
	Instance = db
}

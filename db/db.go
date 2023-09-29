package db

import (
	"log"
	"server/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func Init() {
	db, err := gorm.Open(mysql.Open(config.MYSQL_DSN), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil || db == nil {
		log.Fatalf("DB error: %v", err)
	}
	Instance = db
}

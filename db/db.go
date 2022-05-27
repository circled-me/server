package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB

func Init() {
	dsn := "root:@tcp(127.0.0.1:3306)/circled?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil || db == nil {
		panic(err)
	}
	Instance = db
}

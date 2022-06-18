package models

import (
	"math/rand"
	"server/db"
	"time"
)

func Init() {
	// Seed the random number generator - required for User.Salt
	rand.Seed(time.Now().UnixNano())

	db.Instance.AutoMigrate(&Album{})
	db.Instance.AutoMigrate(&Asset{})
	db.Instance.AutoMigrate(&User{})
	db.Instance.AutoMigrate(&Comment{})
	db.Instance.AutoMigrate(&Grant{})
	db.Instance.AutoMigrate(&Group{})
	db.Instance.AutoMigrate(&Invitation{})
	db.Instance.AutoMigrate(&Like{})
	db.Instance.AutoMigrate(&Post{})
	db.Instance.AutoMigrate(&GroupPost{})
	db.Instance.AutoMigrate(&GroupUser{})

	// b := Bucket{
	// 	Name:        "disk1",
	// 	StorageType: StorageTypeFile,
	// 	Path:        "/Users/nikolay/disk1",
	// }
	// panic(b.Create())
}

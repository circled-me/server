package models

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"server/db"
	"server/storage"
	"server/utils"

	"github.com/pquerna/otp"
)

type User struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   int
	UpdatedAt   int
	CreatedByID *uint64
	CreatedBy   *User
	Name        string        `gorm:"type:varchar(100)"`
	Email       string        `gorm:"type:varchar(150);index:uniq_email,unique"`
	Password    string        `gorm:"type:varchar(256)"` // actually with SHA512 - 128 hex chars is enough (64 bytes output)
	PassSalt    string        `gorm:"type:varchar(200)"`
	TotpToken   string        `gorm:"type:varchar(200)"`
	TotpAlgo    otp.Algorithm `gorm:"type:tinyint(1)"`
	TotpXOR     uint32        `gorm:"type:int unsigned"`
	Grants      []Grant
	BucketID    *uint64
	Bucket      storage.Bucket `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

const saltSize = 60

func randSalt() string {
	b := make([]byte, saltSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func UserCreate(name, email, plainTextPassword string) (u User, err error) {
	// TODO: Be able to pass different storage bucket
	storage := storage.GetDefaultStorage()

	u.Email = email
	u.Name = name
	u.PassSalt = randSalt()
	u.Password = utils.Sha512String(plainTextPassword + u.PassSalt)
	if storage != nil {
		u.BucketID = &storage.GetBucket().ID
	}
	return u, db.Instance.Create(&u).Error
}

func UserLogin(email, plainTextPassword string) (u User, err error) {
	result := db.Instance.Preload("Grants").First(&u, "email = ?", email)
	if result.Error != nil {
		return User{}, errors.New("incorrect email or password")
	}
	if u.Password != utils.Sha512String(plainTextPassword+u.PassSalt) {
		return User{}, errors.New("incorrect email or password")
	}
	return u, nil
}

func (u *User) GetPermissions() []int {
	permissions := []int{}
	for _, grant := range u.Grants {
		permissions = append(permissions, int(grant.Permission))
	}
	return permissions
}

func (u *User) HasPermission(required Permission) bool {
	for _, permission := range u.Grants {
		if permission.Permission == required {
			return true
		}
	}
	return false
}

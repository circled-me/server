package models

import (
	"errors"
	"math/rand"
	"server/db"
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
	PassSalt    string        `gorm:"type:varchar(100)"`
	TotpToken   string        `gorm:"type:varchar(200)"`
	TotpAlgo    otp.Algorithm `gorm:"type:tinyint(1)"`
	TotpXOR     uint32        `gorm:"type:int unsigned"`
	Grants      []Grant
	// Buckets     []Bucket
}

var saltAlphabet = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

const saltSize = 100

func randSalt(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = saltAlphabet[rand.Intn(len(saltAlphabet))]
	}
	return string(b)
}

func UserCreate(name, email, plainTextPassword string) (u User, err error) {
	u.Email = email
	u.Name = name
	u.PassSalt = randSalt(saltSize)
	u.Password = utils.Sha512String(plainTextPassword + u.PassSalt)
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

func (u *User) GetPermissionsArray() []int {
	permissions := []int{}
	for _, grant := range u.Grants {
		permissions = append(permissions, int(grant.Permission))
	}
	return permissions
}

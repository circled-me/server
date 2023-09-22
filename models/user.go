package models

import (
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
	CreatedBy   *User         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name        string        `gorm:"type:varchar(100)"`
	Email       string        `gorm:"type:varchar(150);index:uniq_email,unique"` // TODO: rename Email to Login
	Password    string        `gorm:"type:varchar(128)"`
	PassSalt    string        `gorm:"type:varchar(200)"`
	TotpToken   string        `gorm:"type:varchar(200)"`
	TotpAlgo    otp.Algorithm `gorm:"type:tinyint(1)"`
	TotpXOR     uint32        `gorm:"type:int unsigned"`
	Grants      []Grant       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	BucketID    *uint64
	Bucket      storage.Bucket `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PushToken   string         `gorm:"type:varchar(128)"`

	// Settings
	Quota        int64 `gorm:"not null"` // in MB
	VideoSetting uint8 `gorm:"not null"`
	// ImageProcessing uint8 `gorm:"not null"` // 0 - no, 1 - always to JPEG
}

const (
	saltSize = 60

	VideoSettingConvert = 0
	VideoSettingSkip    = 1
)

func UserCreate(name, email, plainTextPassword string) (u User, err error) {
	// TODO: Be able to pass different storage bucket
	storage := storage.GetDefaultStorage()

	u.Email = email
	u.Name = name
	u.PassSalt = utils.RandSalt(saltSize)
	u.Password = utils.Sha512String(plainTextPassword + u.PassSalt)
	if storage != nil {
		u.BucketID = &storage.GetBucket().ID
	}
	return u, db.Instance.Create(&u).Error
}

func (u *User) SetNewPushToken() {
	u.PushToken = utils.Sha512String(u.Email + utils.RandSalt(saltSize))
	db.Instance.Model(u).Update("push_token", u.PushToken)
}

func (u *User) SetPassword(plainTextPassword string) {
	u.PassSalt = utils.RandSalt(saltSize)
	u.Password = utils.Sha512String(plainTextPassword + u.PassSalt)
}

func UserLogin(email, plainTextPassword string) (u User, success bool) {
	result := db.Instance.Preload("Grants").First(&u, "email = ?", email)
	if result.Error != nil {
		return User{}, false
	}
	if u.Password != utils.Sha512String(plainTextPassword+u.PassSalt) {
		return User{}, false
	}
	return u, true
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

func (u *User) HasPermissions(required []Permission) bool {
	for _, permission := range required {
		if !u.HasPermission(permission) {
			return false
		}
	}
	return true
}

// GetUsage returns the usage for the current bucket (only)
func (u *User) GetUsage() (used, quota int64) {
	result := int64(-1)
	if err := db.Instance.Raw("select ifnull(sum(size+thumb_size), 0) from assets where user_id=? and bucket_id=? and deleted=0", u.ID, u.BucketID).Scan(&result).Error; err != nil {
		return -1, 0
	}
	return result / 1024 / 1024, u.Quota
}

package models

type Permission uint8

const (
	PermissionNone            Permission = 0
	PermissionAdmin           Permission = 1
	PermissionPhotoBackup     Permission = 2
	PermissionCanCreateGroups Permission = 3
	PermissionCanInvite       Permission = 4
)

type Grant struct {
	ID         uint64 `gorm:"primaryKey"`
	CreatedAt  int64
	GrantorID  uint64
	Grantor    User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserID     uint64     `gorm:"index:user_permission,unique"`
	User       User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Permission Permission `gorm:"index:user_permission,unique"`
}

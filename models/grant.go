package models

type Permission uint8

const (
	PermissionNone            Permission = 0
	PermissionAdmin           Permission = 1
	PermissionPhotoBackup     Permission = 2 // photo backup and albums
	PermissionCanCreateGroups Permission = 3
	PermissionCanInvite       Permission = 4 // can invite new users to groups
)

type Grant struct {
	ID         uint64 `gorm:"primaryKey"`
	CreatedAt  int64
	GrantorID  uint64
	Grantor    User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserID     uint64     `gorm:"index:user_permission,unique"`
	User       User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Permission Permission `gorm:"index:user_permission,unique"`
	// In case of `PermissionAdmin` and `PermissionCanInvite` this could be the ID of a Group
	// Subject *uint64 `gorm:"null;default null"`
}

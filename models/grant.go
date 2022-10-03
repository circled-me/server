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
	Grantor    User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserID     uint64
	User       User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Permission Permission
	Subject    uint64 // The object that this permission is valid for (can be 0/nil, e.g. for PermissionAdmin)
}

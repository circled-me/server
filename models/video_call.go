package models

import (
	"server/db"
	"server/utils"
)

type VideoCall struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt int64
	UserID    uint64 `gorm:"index:video_call_user;index:uniq_user_group,unique,priority:1"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupID   uint64 `gorm:"index:video_call_group;index:uniq_user_group,unique,priority:2"`
	ExpiresAt int64
}

func NewVideoCall(userID uint64, groupID uint64, expiresAt int64) (vc VideoCall, err error) {
	vc = VideoCall{
		ID:        utils.Rand8BytesToBase62(),
		UserID:    userID,
		GroupID:   groupID,
		ExpiresAt: expiresAt,
	}
	err = db.Instance.Create(&vc).Error
	return
}

func VideoCallForUser(userID uint64) (vc VideoCall, err error) {
	err = db.Instance.
		Where("user_id = ?", userID).
		First(&vc).
		Error
	if vc.ID == "" {
		return NewVideoCall(userID, 0, 0)
	}
	return
}

func VideoCallByID(id string) (vc VideoCall, err error) {
	err = db.Instance.
		Where("id = ?", id).
		First(&vc).
		Error
	return
}

package models

import "server/utils"

type UploadRequest struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int64
	FixedIP   string
	UserID    uint64 `gorm:"not null"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token     string `gorm:"type:varchar(100);index:uniq_token,unique"`
}

// AssetRemoteIDPrefix is the RemoteID prefix of all assets created through this request
func (r *UploadRequest) AssetRemoteIDPrefix() string {
	prefix := r.Token
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	return prefix + "_"
}

func NewUploadRequest(userID uint64) UploadRequest {
	return UploadRequest{
		UserID: userID,
		Token:  utils.Rand16BytesToBase62(),
	}
}

package models

import "server/db"

type Group struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   int64
	UpdatedAt   int64
	CreatedByID uint64
	CreatedBy   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name        string `gorm:"type:varchar(300);unique"`
}

func LoadGroupUserIDs(groupID uint64) map[uint64]string {
	result := map[uint64]string{}
	rows, err := db.Instance.
		Table("group_users").
		Joins("join users on users.id=user_id").
		Select("user_id, push_token").
		Where("group_id = ?", groupID).
		Rows()
	if err != nil {
		return result
	}
	id := uint64(0)
	token := ""
	for rows.Next() {
		if err = rows.Scan(&id, &token); err != nil {
			continue
		}
		result[id] = token
	}
	rows.Close()
	return result
}

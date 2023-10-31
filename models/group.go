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

func LoadGroupUserIDs(groupID uint64) []uint64 {
	result := []uint64{}
	rows, err := db.Instance.
		Table("group_users").
		Select("user_id").
		Where("group_id = ?", groupID).
		Rows()
	if err != nil {
		return result
	}
	for rows.Next() {
		id := uint64(0)
		if err = rows.Scan(&id); err != nil {
			continue
		}
		result = append(result, id)
	}
	rows.Close()
	return result
}

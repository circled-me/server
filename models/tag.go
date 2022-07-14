package models

type Tag struct {
	ID    uint64 `gorm:"primaryKey"`
	Name  string `gorm:"type:varchar(250);index:name_value,unique,priority:1"`
	Value string `gorm:"type:varchar(250);index:name_value,unique,priority:2"`
}

package models

type Place struct {
	ID      uint64 `gorm:"primaryKey"`
	Area    string `gorm:"type:varchar(100);index:uniq_place,unique,priority:3;not null"`
	City    string `gorm:"type:varchar(100);index:uniq_place,unique,priority:2;not null"`
	Country string `gorm:"type:varchar(100);index:uniq_place,unique,priority:1;not null"`
}

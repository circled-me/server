package models

type Person struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int64  `gorm:""`
	UserID    uint64 `gorm:"index:uniq_user_person,unique;priority:1"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Name      string `gorm:"type:varchar(300);index:uniq_user_person,unique;priority:2"`
}

// TableName overrides the table name
func (Person) TableName() string {
	return "people"
}

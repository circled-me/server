package models

type GroupMessage struct {
	ID          uint64 `gorm:"primaryKey;index:user_urder,priority:2" json:"id"`
	GroupID     uint64 `gorm:"index:group_order,unique,priority:1" json:"group_id"`
	ServerStamp int64  `gorm:"index:group_order,unique,priority:2" json:"server_stamp"`
	ClientStamp int64  `json:"client_stamp"`
	Group       Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	UserID      uint64 `gorm:"index:group_order,unique,priority:3;index:user_urder,priority:1" json:"user_id"`
	User        User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	UserName    string `gorm:"-" json:"user_name"`
	Content     string `gorm:"type:varchar(5000)" json:"content"`
}

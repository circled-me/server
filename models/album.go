package models

type Album struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       uint64 `gorm:"not null;index:user_album_created,priority:1;"`
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	HeroAssetID  *uint64
	HeroAsset    Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreatedAt    int64  `gorm:"index:user_album_created,priority:2"`
	Name         string `gorm:"type:varchar(300)"`
	Hidden       bool   `gorm:"not null;default 0"`
	Contributors []AlbumContributor
}

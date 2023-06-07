package models

type AlbumAsset struct {
	ID        uint64 `gorm:"primaryKey"` // not really needed, but GORM cannot have a foreign key to a composite primary key, so here it is
	CreatedAt int64  `gorm:"index:album_order,priority:2"`
	AlbumID   uint64 `gorm:"index:uniq_album_asset,unique;index:album_order,priority:1"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID   uint64 `gorm:"index:uniq_album_asset,unique;"`
	Asset     Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

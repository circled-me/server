package models

type Face struct {
	ID        uint64 `gorm:"primaryKey"` // not really needed, but GORM cannot have a foreign key to a composite primary key, so here it is
	CreatedAt int64  `gorm:""`
	AssetID   uint64 `gorm:"index:uniq_asset_face,unique;priority:1"`
	Asset     Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Num       int    `gorm:"index:uniq_asset_face,unique;"`
	Location  string `gorm:"type:varchar(1024)"`  // Contains JSON array, e.g. [left,top,right,bottom]
	Encoding  string `gorm:"type:varchar(65535)"` // Contains JSON array of 128 floats
}

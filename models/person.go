package models

type Person struct {
	ID      uint64 `gorm:"primaryKey"`
	AssetID uint64
	Asset   Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Vector  []byte `gorm:"type:blob"`
	RectX1  uint16
	RectY1  uint16
	RectX2  uint16
	RectY2  uint16
}

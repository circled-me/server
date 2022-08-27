package models

const (
	StatusTodo       = 0
	StatusProcessed  = 1
	StatusProcessing = 100
	StatusError      = 101
)

type PostProcessing struct {
	ID        uint64 `gorm:"primaryKey"` // Corresponds to the Asset.ID
	StartedAt int64  // UNIX timestamp
	// CompressionStatus        uint8
	// TODO face recognition, more...
}

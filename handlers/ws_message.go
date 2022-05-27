package handlers

type WSMessageType uint8

const (
	WSMessageTypeAssetUpload WSMessageType = iota
)

type WSMessage struct {
	Type WSMessageType `json:"type"`
	Size uint32
}

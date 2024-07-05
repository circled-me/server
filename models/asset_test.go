package models

import (
	"reflect"
	"server/storage"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

func TestAsset_GetCreatedTimeInLocation(t *testing.T) {
	type fields struct {
		ID                  uint64
		UserID              uint64
		RemoteID            string
		CreatedAt           int64
		UpdatedAt           int64
		Size                int64
		ThumbSize           int64
		User                User
		GroupID             *uint64
		Group               Group
		BucketID            uint64
		Bucket              storage.Bucket
		PlaceID             *uint64
		Place               Place
		Name                string
		MimeType            string
		GpsLat              *float64
		GpsLong             *float64
		Favourite           bool
		Deleted             bool
		Width               uint16
		Height              uint16
		ThumbWidth          uint16
		ThumbHeight         uint16
		Duration            uint32
		Path                string
		ThumbPath           string
		PresignedUntil      int64
		PresignedURL        string
		PresignedThumbUntil int64
		PresignedThumbURL   string
	}
	CST, _ := time.LoadLocation("Asia/Shanghai")
	tests := []struct {
		name   string
		fields fields
		want   time.Time
	}{
		{
			name: "Asia/Shanghai", // CST
			fields: fields{
				CreatedAt: 1696258800,
				GpsLat:    aws.Float64(39.9254474),
				GpsLong:   aws.Float64(116.3870752),
			},
			want: time.Unix(1696258800, 0).Local().In(CST),
		},
		{
			name: "Local", // when no GPS coords
			fields: fields{
				CreatedAt: 1696258800,
			},
			want: time.Unix(1696258800, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Asset{
				ID:                  tt.fields.ID,
				UserID:              tt.fields.UserID,
				RemoteID:            tt.fields.RemoteID,
				CreatedAt:           tt.fields.CreatedAt,
				UpdatedAt:           tt.fields.UpdatedAt,
				Size:                tt.fields.Size,
				ThumbSize:           tt.fields.ThumbSize,
				User:                tt.fields.User,
				GroupID:             tt.fields.GroupID,
				Group:               tt.fields.Group,
				BucketID:            tt.fields.BucketID,
				Bucket:              tt.fields.Bucket,
				PlaceID:             tt.fields.PlaceID,
				Place:               tt.fields.Place,
				Name:                tt.fields.Name,
				MimeType:            tt.fields.MimeType,
				GpsLat:              tt.fields.GpsLat,
				GpsLong:             tt.fields.GpsLong,
				Favourite:           tt.fields.Favourite,
				Deleted:             tt.fields.Deleted,
				Width:               tt.fields.Width,
				Height:              tt.fields.Height,
				ThumbWidth:          tt.fields.ThumbWidth,
				ThumbHeight:         tt.fields.ThumbHeight,
				Duration:            tt.fields.Duration,
				Path:                tt.fields.Path,
				ThumbPath:           tt.fields.ThumbPath,
				PresignedUntil:      tt.fields.PresignedUntil,
				PresignedURL:        tt.fields.PresignedURL,
				PresignedThumbUntil: tt.fields.PresignedThumbUntil,
				PresignedThumbURL:   tt.fields.PresignedThumbURL,
			}
			if got := a.GetCreatedTimeInLocation(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Asset.GetCreatedTimeInLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

package faces

import "encoding/json"

const (
	IndexTop    = 0
	IndexRight  = 1
	IndexBottom = 2
	IndexLeft   = 3
)

type (
	FaceBoundaries      [4]int
	FaceBoundariesList  []FaceBoundaries
	FaceEncoding        [128]float64
	FaceEncodingList    []FaceEncoding
	FaceDetectionResult struct {
		Locations FaceBoundariesList `json:"locations"`
		Encodings FaceEncodingList   `json:"encodings"`
	}
)

func toFacesResult(data []byte) (result FaceDetectionResult, err error) {
	// Parse the string
	err = json.Unmarshal(data, &result)
	return result, err
}

func (l *FaceBoundaries) ToJSONString() string {
	data, _ := json.Marshal(l)
	return string(data)
}

func (e *FaceEncoding) ToJSONString() string {
	data, _ := json.Marshal(e)
	return string(data)
}

package handlers

type Response struct {
	Error string `json:"error"`
}

type MultiResponse struct {
	Error  string   `json:"error"`
	Failed []uint64 `json:"failed"`
}

var (
	// Predefined errors
	OKResponse       = Response{}
	NopeResponse     = Response{"nope"}
	Nope2Response    = Response{"no no"}
	Nope3Response    = Response{"no no no"}
	DBError1Response = Response{"DB Error 1"}
	DBError2Response = Response{"DB Error 2"}
	DBError3Response = Response{"DB Error 3"}
	DBError4Response = Response{"DB Error 4"}
	OKMultiResponse  = MultiResponse{}
)

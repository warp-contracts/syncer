package responses

type Upload struct {
	Id                  string   `json:"id"`
	Timestamp           uint64   `json:"timestamp"`
	Winc                string   `json:"winc"`
	Version             string   `json:"version"`
	DeadlineHeight      uint64   `json:"deadlineHeight"`
	DataCaches          []string `json:"dataCaches"`
	FastFinalityIndexes []string `json:"fastFinalityIndexes"`
	Public              string   `json:"public"`
	Signature           string   `json:"signature"`
	Owner               string   `json:"owner"`
}

package request

type GetInteractions struct {
	Start  uint `json:"start"`
	End    uint `json:"end"`
	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
}

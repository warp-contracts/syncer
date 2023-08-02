package request

type GetInteractions struct {
	SrcIds []string `json:"src_ids"`
	Start  uint     `json:"start"`
	End    uint     `json:"end"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

package responses

type GetNonce struct {
	Address string `json:"address"`
	Nonce   uint64 `json:"nonce"`
}

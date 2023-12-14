package responses

type Upload struct {
	SequencerTxHash string `json:"sequencer_tx_hash"`
	DataItemId      string `json:"data_item_id"`
}

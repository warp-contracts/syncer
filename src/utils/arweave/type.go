package arweave

type NetworkInfo struct {
	Network          string `json:"network"`
	Version          int64  `json:"version"`
	Release          int64  `json:"release"`
	Height           int64  `json:"height"`
	Current          string `json:"current"`
	Blocks           int64  `json:"blocks"`
	Peers            int64  `json:"peers"`
	QueueLength      int64  `json:"queue_length"`
	NodeStateLatency int64  `json:"node_state_latency"`
}

type Block struct {
	Nonce                    string        `json:"nonce"`
	PreviousBlock            string        `json:"previous_block"`
	Timestamp                int64         `json:"timestamp"`
	LastRetarget             int64         `json:"last_retarget"`
	Height                   int64         `json:"height"`
	Diff                     interface{}   `json:"diff"`
	Hash                     string        `json:"hash"`
	IndepHash                string        `json:"indep_hash"`
	Txs                      []string      `json:"txs"`
	TxRoot                   string        `json:"tx_root"`
	TxTree                   interface{}   `json:"tx_tree"`
	HashList                 interface{}   `json:"hash_list"`
	HashListMerkle           string        `json:"hash_list_merkle"`
	WalletList               string        `json:"wallet_list"`
	RewardAddr               string        `json:"reward_addr"`
	Tags                     []interface{} `json:"tags"`
	RewardPool               interface{}   `json:"reward_pool"` // always string
	WeaveSize                interface{}   `json:"weave_size"`  // always string
	BlockSize                interface{}   `json:"block_size"`  // always string
	CumulativeDiff           interface{}   `json:"cumulative_diff"`
	SizeTaggedTxs            interface{}   `json:"size_tagged_txs"`
	Poa                      POA           `json:"poa"`
	UsdToArRate              []string      `json:"usd_to_ar_rate"`
	ScheduledUsdToArRate     []string      `json:"scheduled_usd_to_ar_rate"`
	Packing25Threshold       string        `json:"packing_2_5_threshold"`
	StrictDataSplitThreshold string        `json:"strict_data_split_threshold"`
}

type POA struct {
	Option   string `json:"option"`
	TxPath   string `json:"tx_path"`
	DataPath string `json:"data_path"`
	Chunk    string `json:"chunk"`
}

type Transaction struct {
	Format    int    `json:"format"`
	ID        string `json:"id"`
	LastTx    string `json:"last_tx"`
	Owner     string `json:"owner"` // utils.Base64Encode(wallet.PubKey.N.Bytes())
	Tags      []Tag  `json:"tags"`
	Target    string `json:"target"`
	Quantity  string `json:"quantity"`
	Data      string `json:"data"` // base64.encode
	DataSize  string `json:"data_size"`
	DataRoot  string `json:"data_root"`
	Reward    string `json:"reward"`
	Signature string `json:"signature"`

	// Computed when needed.
	Chunks *Chunks `json:"-"`
}

type Tag struct {
	Name  Base64String `json:"name" avro:"name"`
	Value Base64String `json:"value" avro:"value"`
}

type Chunks struct {
	DataRoot []byte   `json:"data_root"`
	Chunks   []Chunk  `json:"chunks"`
	Proofs   []*Proof `json:"proofs"`
}

type Chunk struct {
	DataHash     []byte
	MinByteRange int
	MaxByteRange int
}

type Proof struct {
	Offest int
	Proof  []byte
}

package report

type Report struct {
	Syncer      *SyncerReport      `json:"syncer,omitempty"`
	Bundler     *BundlerReport     `json:"bundler,omitempty"`
	NetworkInfo *NetworkInfoReport `json:"network_info,omitempty"`
}

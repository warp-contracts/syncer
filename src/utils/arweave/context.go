package arweave

type value string

const (
	// Setting this flag disables retrying request with peers
	ContextDisablePeers = value("disablePeers")
)

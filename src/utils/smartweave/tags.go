package smartweave

import "regexp"

const (
	TagAppName         = "App-Name"
	TagAppNameValue    = "SmartWeaveAction"
	TagAppVersion      = "App-Version"
	TagContractTxId    = "Contract"
	TagInput           = "Input"
	TagContentType     = "Content-Type"
	TagContractSrcTxId = "Contract-Src"
	TagSDK             = "SDK"
	TagMinFee          = "Min-Fee"
)

var (
	TagContractTxIdRegex = regexp.MustCompile("^[a-zA-Z0-9_-]{43}$")
)

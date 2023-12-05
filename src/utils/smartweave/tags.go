package smartweave

import "regexp"

// SmartWeave protocol tag names
const (
	TagAppName         = "App-Name"
	TagAppVersion      = "App-Version"
	TagContractTxId    = "Contract"
	TagInput           = "Input"
	TagInputFormat     = "Input-Format"
	TagContentType     = "Content-Type"
	TagContractSrcTxId = "Contract-Src"
	TagSDK             = "SDK"
	TagMinFee          = "Min-Fee"
)

// SmartWeave protocol tag values
const (
	TagAppNameValue         = "SmartWeaveAction"
	TagInputFormatTagValue  = "tag"
	TagInputFormatDataValue = "data"
)

var (
	TagContractTxIdRegex = regexp.MustCompile("^[a-zA-Z0-9_-]{43}$")
)

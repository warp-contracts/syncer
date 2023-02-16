package model

// JSON message sent through the notification channel
type BundleItemNotification struct {
	BundleItem BundleItem `json:"item"`
	IsEmpty    bool       `json:"is_empty"`
}

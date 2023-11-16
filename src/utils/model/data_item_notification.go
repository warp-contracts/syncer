package model

// JSON message sent through the notification channel
type DataItemNotification struct {
	DataItemId string `json:"id"`
}

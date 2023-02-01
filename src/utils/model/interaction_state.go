package model

import "database/sql/driver"

type InteractionState string

const (
	InteractionStatePending   InteractionState = "PENDING"
	InteractionStateUploading InteractionState = "UPLOADING"
	InteractionStateUploaded  InteractionState = "UPLOADED"
	InteractionStateConfirmed InteractionState = "CONFIRMED"
)

func (self *InteractionState) Scan(value interface{}) error {
	*self = InteractionState(value.([]byte))
	return nil
}

func (self InteractionState) Value() (driver.Value, error) {
	return string(self), nil
}

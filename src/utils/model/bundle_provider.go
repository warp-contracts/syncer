package model

import "database/sql/driver"

type BundlingService string

const (
	BundlingServiceIrys  BundlingService = "Irys"
	BundlingServiceTurbo BundlingService = "Turbo"
)

func (self *BundlingService) Scan(value interface{}) error {
	*self = BundlingService(value.(string))
	return nil
}

func (self BundlingService) Value() (driver.Value, error) {
	return string(self), nil
}

package model

import "database/sql/driver"

// CREATE TYPE bundle_state AS ENUM ('PENDING', 'UPLOADING', 'UPLOADED', 'ON_BUNDLER', 'ON_ARWEAVE');
type BundleState string

const (
	BundleStatePending   BundleState = "PENDING"
	BundleStateUploading BundleState = "UPLOADING"
	BundleStateUploaded  BundleState = "UPLOADED"
	BundleStateOnBundler BundleState = "ON_BUNDLER"
	BundleStateOnArweave BundleState = "ON_ARWEAVE"
)

func (self *BundleState) Scan(value interface{}) error {
	*self = BundleState(value.(string))
	return nil
}

func (self BundleState) Value() (driver.Value, error) {
	return string(self), nil
}

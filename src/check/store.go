package check

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"gorm.io/gorm"
)

// Periodically saves the states
// SinkTask handles caching data and periodically calling flush function
type Store struct {
	*task.SinkTask[int]

	db *gorm.DB
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.SinkTask = task.NewSinkTask[int](config, "store").
		WithOnFlush(5*time.Second, self.save).
		WithBatchSize(50)
	return
}

func (self *Store) WithDB(db *gorm.DB) *Store {
	self.db = db
	return self
}

func (self *Store) WithInputChannel(input chan int) *Store {
	self.SinkTask = self.SinkTask.WithInputChannel(input)
	return self
}

func (self *Store) save(ids []int) error {
	err := self.db.Model(&model.BundleItem{}).
		Where("id IN ?", ids).
		Update("state", model.BundleStateOnBundler).
		Error
	if err != nil {
		return err
	}
	return nil
}
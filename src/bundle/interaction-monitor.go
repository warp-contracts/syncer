package bundle

import (
	"syncer/src/utils/config"
	"syncer/src/utils/task"
	"time"
)

type InteractionMonitor struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
func NewInteractionMonitor(config *config.Config) (self *InteractionMonitor, err error) {
	self = new(InteractionMonitor)
	self.Task = task.NewTask(config, "interaction-monitor").
		WithSubtaskFunc(self.monitorInteractions)
	return
}

func (self *InteractionMonitor) monitorInteractions() error {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-self.StopChannel:
			self.Log.Debug("Stop monitoring interactions")

			return nil
		case <-ticker.C:
			self.Log.Debug("Tick")
		}
	}
}

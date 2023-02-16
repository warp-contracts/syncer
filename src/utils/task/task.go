package task

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/sirupsen/logrus"
)

// Boilerplate for running long lived tasks.
type Task struct {
	Config *config.Config
	Log    *logrus.Entry

	// Stopping
	IsStopping    *atomic.Bool
	StopChannel   chan bool
	stopOnce      *sync.Once
	stopWaitGroup sync.WaitGroup

	// Context active as long as there's anything running in the task.
	// Used outside the task.
	CtxRunning    context.Context
	cancelRunning context.CancelFunc

	// Context cancelled when Stop() is called.
	// Used inside the task
	Ctx    context.Context
	cancel context.CancelFunc

	// Workers that perform the task
	Workers *workerpool.WorkerPool

	// Callbacks
	onBeforeStart []func() error
	onStop        []func()
	onAfterStop   []func()
	subtasksFunc  []func() error
	subtasks      []*Task
}

func NewTask(config *config.Config, name string) (self *Task) {
	self = new(Task)
	self.Log = logger.NewSublogger(name)
	self.Config = config

	// Context cancelled when Stop() is called
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Context active as long as there's anything running in the task
	self.CtxRunning, self.cancelRunning = context.WithCancel(context.Background())
	self.CtxRunning = common.SetConfig(self.Ctx, config)

	// Stopping
	self.stopOnce = &sync.Once{}
	self.IsStopping = &atomic.Bool{}
	self.stopWaitGroup = sync.WaitGroup{}
	self.StopChannel = make(chan bool, 1)

	return
}

func (self *Task) WithOnBeforeStart(f func() error) *Task {
	self.onBeforeStart = append(self.onBeforeStart, f)
	return self
}

func (self *Task) WithOnAfterStop(f func()) *Task {
	self.onAfterStop = append(self.onAfterStop, f)
	return self
}

func (self *Task) WithOnStop(f func()) *Task {
	self.onStop = append(self.onStop, f)
	return self
}

func (self *Task) WithSubtask(t *Task) *Task {
	// Ensure context will be cancelled after all kinds of subtasks finish
	t = t.WithOnBeforeStart(func() error {
		self.stopWaitGroup.Add(1)
		return nil
	}).WithOnAfterStop(func() {
		self.stopWaitGroup.Done()
	})
	self.subtasks = append(self.subtasks, t)
	return self
}

func (self *Task) WithSubtaskFunc(f func() error) *Task {
	self.subtasksFunc = append(self.subtasksFunc, f)
	return self
}

func (self *Task) WithPeriodicSubtaskFunc(period time.Duration, f func() error) *Task {
	self.subtasksFunc = append(self.subtasksFunc, func() error {
		var timer *time.Timer
		run := func() error {
			// Setup waiting before the next check
			defer func() { timer = time.NewTimer(period) }()
			return f()
		}

		var err error
		for {
			err = run()
			if err != nil {
				return err
			}

			select {
			case <-self.StopChannel:
				self.Log.Debug("Task stopped")
				return nil
			case <-timer.C:
				// pass through
			}
		}
	})
	return self
}

func (self *Task) WithWorkerPool(maxWorkers int) *Task {
	self.Workers = workerpool.New(maxWorkers)
	return self.WithOnAfterStop(func() {
		self.Workers.StopWait()
	})
}

func (self *Task) run(subtask func() error) {
	self.stopWaitGroup.Add(1)
	go func() {
		defer func() {
			self.stopWaitGroup.Done()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.Log.WithError(err).Error("Panic. Stopping.")

				panic(p)
			}
		}()
		err := subtask()
		if err != nil {
			self.Log.WithError(err).Error("Subtask failed")
		}
	}()
}

func (self *Task) Start() (err error) {
	// Run callbacks
	for _, cb := range self.onBeforeStart {
		err = cb()
		if err != nil {
			return
		}
	}

	// Start subtasks
	for _, subtask := range self.subtasks {
		err = subtask.Start()
		if err != nil {
			return
		}
	}

	// Start subtasks that are plain functions
	for _, subtask := range self.subtasksFunc {
		self.run(subtask)
	}

	// Gorouting that will cancel the context
	go func() {
		// Infinite wait, assuming all subtasks will eventually close using the StopChannel
		self.stopWaitGroup.Wait()

		// Run hooks
		for _, cb := range self.onAfterStop {
			cb()
		}

		// Inform that task doesn't run anymore
		self.cancelRunning()
	}()

	return nil
}

func (self *Task) Stop() {
	self.Log.Info("Stopping...")
	self.stopOnce.Do(func() {
		// Stop subtasks
		for _, subtask := range self.subtasks {
			subtask.Stop()
		}

		// Signals that we're stopping
		close(self.StopChannel)

		// Inform child context that we're stopping
		self.cancel()

		// Mark that we're stopping
		self.IsStopping.Store(true)

		// Run hooks
		for _, cb := range self.onStop {
			cb()
		}
	})
}

func (self *Task) StopWait() {
	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), self.Config.StopTimeout)
	defer cancel()

	self.Stop()

	select {
	case <-ctx.Done():
		self.Log.Error("Timeout reached, failed to stop")
	case <-self.Ctx.Done():
		self.Log.Info("Task finished")
	}
}

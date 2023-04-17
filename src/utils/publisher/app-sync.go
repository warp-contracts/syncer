package publisher

import (
	"encoding"
	"encoding/json"
	"fmt"
	"syncer/src/utils/config"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"time"

	appsync "github.com/sony/appsync-client-go"
	"github.com/sony/appsync-client-go/graphql"
)

// Forwards messages to Redis
type AppSyncPublisher[In encoding.BinaryMarshaler] struct {
	*task.Task

	monitor monitoring.Monitor

	client      *appsync.Client
	channelName string
	input       chan In
}

func NewAppSyncPublisher[In encoding.BinaryMarshaler](config *config.Config, name string) (self *AppSyncPublisher[In]) {
	self = new(AppSyncPublisher[In])

	self.Task = task.NewTask(config, name).
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.AppSync.MaxWorkers, config.AppSync.MaxQueueSize)

	// Init AppSync client
	gqlClient := graphql.NewClient(config.AppSync.Url,
		graphql.WithAPIKey(config.AppSync.Token),
		graphql.WithTimeout(time.Second*30),
	)

	self.client = appsync.NewClient(appsync.NewGraphQLClient(gqlClient))

	return
}

func (self *AppSyncPublisher[In]) WithInputChannel(v chan In) *AppSyncPublisher[In] {
	self.input = v
	return self
}

func (self *AppSyncPublisher[In]) WithChannelName(v string) *AppSyncPublisher[In] {
	self.channelName = v
	return self
}

func (self *AppSyncPublisher[In]) WithMonitor(monitor monitoring.Monitor) *AppSyncPublisher[In] {
	self.monitor = monitor
	return self
}

func (self *AppSyncPublisher[In]) publish(data []byte) (err error) {
	mutation := `mutation Publish($data: AWSJSON!, $name: String!) {
	  publish(data: $data, name: $name) {
		data
		name
	  }
	}`

	self.Log.Debug("A")

	variables := json.RawMessage(fmt.Sprintf(`{"name":"%s","data":%s}`, self.channelName, data))
	self.Log.Debug("B")
	response, err := self.client.Post(graphql.PostRequest{
		Query:     mutation,
		Variables: &variables,
	})
	self.Log.Debug("C")
	if err != nil {
		self.Log.Debug("D")
		return err
	}

	self.Log.Debug("E")
	body := new(string)
	err = response.DataAs(body)
	self.Log.Debug("F")
	if err != nil {
		self.Log.Debug("G")
		return err
	}

	self.Log.Debug("H")
	self.Log.WithField("code", *response.StatusCode).WithField("body", body).Info("AppSync response")
	return nil
}

func (self *AppSyncPublisher[In]) run() (err error) {
	for data := range self.input {
		data := data
		self.SubmitToWorker(func() {
			self.Log.Debug("App sync publish...")
			defer self.Log.Debug("...App sync publish done")

			// Serialize to JSON
			self.Log.Debug("1")
			jsonData, err := data.MarshalBinary()
			if err != nil {
				self.Log.WithError(err).Error("Failed to marshal to json")
				return
			}

			self.Log.Debug("2")
			// Retry on failure with exponential backoff
			err = task.NewRetry().
				WithMaxElapsedTime(self.Config.AppSync.BackoffMaxElapsedTime).
				WithMaxInterval(self.Config.AppSync.BackoffMaxInterval).
				WithOnError(func(err error) {
					self.Log.WithError(err).Error("Appsync publish failed")
					self.monitor.GetReport().AppSyncPublisher.Errors.Publish.Inc()
				}).
				Run(func() error {
					self.Log.Debug("3")
					return self.publish(jsonData)
				})

			self.Log.Debug("4")
			if err != nil {
				self.Log.WithError(err).Error("Failed to publish to appsync after retries")
				self.monitor.GetReport().AppSyncPublisher.Errors.PersistentFailure.Inc()
				return
			}

			self.Log.Debug("5")
			self.monitor.GetReport().AppSyncPublisher.State.MessagesPublished.Inc()
		})
	}
	return nil
}

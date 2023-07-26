package sequencer

import (
	"math/rand"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/task"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type EventSource struct {
	*task.Task

	Output      chan string
	isConnected atomic.Bool
	streamId    int
	// Config
	subscribe func(ws *websocket.Conn) error
}

// Maintains a persistent websocket connection to the sequencer
// Gets events that it subscribed for
func NewEventSource(config *config.Config, name string) (self *EventSource) {
	self = new(EventSource)

	self.Output = make(chan string)

	self.Task = task.NewWebsocket[](config, name).
		WithUrl(config.Relayer.SequencerUrl).
		WithOnConnected(self.subscribe).
		WithOnDisconnected(self.onDisconnected)

	// '{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='NewBlock'" } }'
}

func (self *EventSource) WithEventName(eventName string) *EventSource {
	self.subscribe = func(ws *websocket.Conn) error {
		self.isConnected = true
		self.streamId = rand.Int()
		args := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "subscribe",
			"id":      self.streamId,
			"params": map[string]interface{}{	
				"query": "tm.event='" + eventName + "'",	
			},
		}
		err := wsjson.Write(self.Ctx, ws, args)
		if err != nil {
			return err
		}

		var resp map[string]interface{}
		err = wsjson.Read(self.Ctx, ws, &resp)
		if err != nil {
			return err
		}

		if resp["id"] != self.streamId {
			return errors.New("invalid response id")
		}

		return nil
	}

	return self
}

func (self *EventSource) onDisconnected() {
	self.isConnected = false
}



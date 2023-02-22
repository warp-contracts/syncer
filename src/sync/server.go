package sync

import (
	"context"
	"net/http"
	"syncer/src/utils/config"
	"syncer/src/utils/monitor"
	"syncer/src/utils/task"

	"github.com/gin-gonic/gin"
)

// Rest API server, serves monitor counters
type Server struct {
	*task.Task

	httpServer *http.Server
	Router     *gin.Engine

	monitor *monitor.Monitor
}

func NewServer(config *config.Config) (self *Server) {
	self = new(Server)

	self.Task = task.NewTask(config, "server").
		WithSubtaskFunc(self.run).
		WithOnStop(self.stop)

	self.Router = gin.New()

	self.httpServer = &http.Server{
		Addr:    self.Config.RESTListenAddress,
		Handler: self.Router,
	}

	return
}

func (self *Server) WithMonitor(monitor *monitor.Monitor) *Server {
	self.monitor = monitor

	return self
}

func (self *Server) run() (err error) {
	gin.SetMode(gin.DebugMode)

	v1 := self.Router.Group("v1")
	{
		v1.GET("health", self.monitor.OnGet)
	}

	err = self.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		self.Log.WithError(err).Error("Failed to start REST server")
		return
	}
	return nil
}

func (self *Server) stop() {
	ctx, cancel := context.WithTimeout(context.Background(), self.Config.StopTimeout)
	defer cancel()

	err := self.httpServer.Shutdown(ctx)
	if err != nil {
		self.Log.WithError(err).Error("Failed to gracefully shutdown REST server")
		return
	}
}

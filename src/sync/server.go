package sync

import (
	"context"
	"fmt"
	"net/http"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Rest API server
type Server struct {
	Ctx    context.Context
	cancel context.CancelFunc
	config *config.Config
	log    *logrus.Entry

	httpServer *http.Server
	Router     *gin.Engine

	Monitor *Monitor
}

func NewServer(config *config.Config, monitor *Monitor) (self *Server, err error) {
	self = new(Server)
	self.log = logger.NewSublogger("server")
	self.config = config

	self.Ctx, self.cancel = context.WithCancel(context.Background())

	// Setup router
	self.Router = gin.New()

	gin.SetMode(gin.DebugMode)
	v1 := self.Router.Group("v1")
	{
		v1.GET("health", monitor.OnGet)
	}

	self.httpServer = &http.Server{
		Addr:    self.config.RESTListenAddress,
		Handler: self.Router,
	}
	return
}

func (self *Server) Start() {
	go func() {
		defer func() {
			// run() finished, so it's time to cancel Listener's context
			// NOTE: This should be the only place self.Ctx is cancelled
			self.cancel()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in Server. Stopping.")

				// NOTE: Panics in listener are suppressed
				// because syncer is using panics for reporting errors...
				panic(p)
			}
		}()
		self.log.Info("Started REST server")
		if err := self.httpServer.ListenAndServe(); err != nil {
			self.log.WithError(err).Info("REST server finished")
		}
	}()
}

func (self *Server) StopWait() {
	self.log.Info("Stopping server")

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := self.httpServer.Shutdown(ctx)
	if err != nil {
		self.log.WithError(err).Error("Failed to gracefully shutdown REST server")
		return
	}
}

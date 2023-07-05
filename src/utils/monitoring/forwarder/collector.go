package monitor_forwarder

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	// Forwarder
	FinishedHeight     *prometheus.Desc
	L1Interactions     *prometheus.Desc
	L2Interactions     *prometheus.Desc
	BlocksBehindSyncer *prometheus.Desc

	// Redis publisher
	RedisPublishErrors     *prometheus.Desc
	RedisPersistentErrors  *prometheus.Desc
	RedisMessagesPublished *prometheus.Desc
	RedisPoolHits          *prometheus.Desc
	RedisPoolIdleConns     *prometheus.Desc
	RedisPoolMisses        *prometheus.Desc
	RedisPoolStaleConns    *prometheus.Desc
	RedisPoolTimeouts      *prometheus.Desc
	RedisPoolTotalConns    *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		// Run
		UpForSeconds: prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Forwarder
		FinishedHeight:     prometheus.NewDesc("finished_height", "", nil, nil),
		L1Interactions:     prometheus.NewDesc("l1_interactions", "", nil, nil),
		L2Interactions:     prometheus.NewDesc("l2_interactions", "", nil, nil),
		BlocksBehindSyncer: prometheus.NewDesc("blocks_behind_syncer", "", nil, nil),

		// Redis publisher
		RedisPublishErrors:     prometheus.NewDesc("error_redis_publish_errors", "", nil, nil),
		RedisPersistentErrors:  prometheus.NewDesc("error_redis_persistent_errors", "", nil, nil),
		RedisMessagesPublished: prometheus.NewDesc("redis_messages_published", "", nil, nil),
		RedisPoolHits:          prometheus.NewDesc("redis_pool_hits", "", nil, nil),
		RedisPoolIdleConns:     prometheus.NewDesc("redis_pool_idle_conns", "", nil, nil),
		RedisPoolMisses:        prometheus.NewDesc("redis_pool_misses", "", nil, nil),
		RedisPoolStaleConns:    prometheus.NewDesc("redis_pool_stale_conns", "", nil, nil),
		RedisPoolTimeouts:      prometheus.NewDesc("redis_pool_timeouts", "", nil, nil),
		RedisPoolTotalConns:    prometheus.NewDesc("redis_pool_total_conns", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds

	// Forwarder
	ch <- self.FinishedHeight
	ch <- self.L1Interactions
	ch <- self.L2Interactions
	ch <- self.BlocksBehindSyncer

	// Redis publisher
	ch <- self.RedisPublishErrors
	ch <- self.RedisPersistentErrors
	ch <- self.RedisMessagesPublished
	ch <- self.RedisPoolHits
	ch <- self.RedisPoolIdleConns
	ch <- self.RedisPoolMisses
	ch <- self.RedisPoolStaleConns
	ch <- self.RedisPoolTimeouts
	ch <- self.RedisPoolTotalConns
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Forwarder
	ch <- prometheus.MustNewConstMetric(self.FinishedHeight, prometheus.GaugeValue, float64(self.monitor.Report.Forwarder.State.FinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.L1Interactions, prometheus.CounterValue, float64(self.monitor.Report.Forwarder.State.L1Interactions.Load()))
	ch <- prometheus.MustNewConstMetric(self.L2Interactions, prometheus.CounterValue, float64(self.monitor.Report.Forwarder.State.L2Interactions.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlocksBehindSyncer, prometheus.GaugeValue, float64(self.monitor.Report.Forwarder.State.BlocksBehindSyncer.Load()))

	// Redis publisher
	ch <- prometheus.MustNewConstMetric(self.RedisPublishErrors, prometheus.CounterValue, float64(self.monitor.Report.RedisPublisher.Errors.Publish.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPersistentErrors, prometheus.CounterValue, float64(self.monitor.Report.RedisPublisher.Errors.PersistentFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisMessagesPublished, prometheus.CounterValue, float64(self.monitor.Report.RedisPublisher.State.MessagesPublished.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolHits, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolHits.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolIdleConns, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolIdleConns.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolMisses, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolMisses.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolStaleConns, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolStaleConns.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolTimeouts, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolTimeouts.Load()))
	ch <- prometheus.MustNewConstMetric(self.RedisPoolTotalConns, prometheus.GaugeValue, float64(self.monitor.Report.RedisPublisher.State.PoolTotalConns.Load()))
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	nsPlayer  = "player"
	nsIAPI    = "iapi"
	nsProxy   = "proxy"
	nsLbrynet = "lbrynet"
	nsUI      = "ui"
	nsLbrytv  = "lbrytv"

	LabelSource   = "source"
	LabelInstance = "instance"

	LabelNameType  = "type"
	LabelValuePaid = "paid"
	LabelValueFree = "free"
)

var (
	IAPIAuthSuccessDurations = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: nsIAPI,
		Subsystem: "auth",
		Name:      "success_seconds",
		Help:      "Time to successful authentication",
	})
	IAPIAuthFailedDurations = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: nsIAPI,
		Subsystem: "auth",
		Name:      "failed_seconds",
		Help:      "Time to failed authentication response",
	})

	callsSecondsBuckets = []float64{0.005, 0.025, 0.05, 0.1, 0.25, 0.4, 1, 2, 5, 10, 20, 60, 120, 300}

	ProxyCallDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: nsProxy,
			Subsystem: "calls",
			Name:      "total_seconds",
			Help:      "Method call latency distributions",
			Buckets:   callsSecondsBuckets,
		},
		[]string{"method", "endpoint"},
	)
	ProxyCallFailedDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: nsProxy,
			Subsystem: "calls",
			Name:      "failed_seconds",
			Help:      "Failed method call latency distributions",
			Buckets:   callsSecondsBuckets,
		},
		[]string{"method", "endpoint"},
	)

	LbrynetWalletsLoaded = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: nsLbrynet,
		Subsystem: "wallets",
		Name:      "count",
		Help:      "Number of wallets currently loaded",
	}, []string{LabelSource})

	UIBufferCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: nsUI,
		Subsystem: "content",
		Name:      "buffer_count",
		Help:      "Video buffer events",
	})
	UITimeToStart = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: nsUI,
		Subsystem: "content",
		Name:      "time_to_start",
		Help:      "How long it takes the video to start",
		Buckets:   []float64{0.1, 0.25, 0.5, 1, 2, 4, 8, 16, 32},
	})

	LbrytvCallDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: nsLbrytv,
			Subsystem: "calls",
			Name:      "total_seconds",
			Help:      "How long do calls to lbrytv take (end-to-end)",
			Buckets:   callsSecondsBuckets,
		},
		[]string{"path"},
	)

	LbrytvNewUsers = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: nsLbrytv,
		Subsystem: "users",
		Name:      "count",
		Help:      "Total number of new users created in the database",
	})
	LbrytvPurchases = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: nsLbrytv,
		Subsystem: "purchase",
		Name:      "count",
		Help:      "Total number of purchases done",
	})
	LbrytvStreamRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: nsLbrytv,
		Subsystem: "stream",
		Name:      "count",
		Help:      "Total number of stream requests received",
	}, []string{LabelNameType})

	LbrytvDBOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: nsLbrytv,
		Subsystem: "db",
		Name:      "conns_open",
		Help:      "Number of open db connections in the Go connection pool",
	})
	LbrytvDBInUseConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: nsLbrytv,
		Subsystem: "db",
		Name:      "conns_in_use",
		Help:      "Number of in-use db connections in the Go connection pool",
	})
	LbrytvDBIdleConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: nsLbrytv,
		Subsystem: "db",
		Name:      "conns_idle",
		Help:      "Number of idle db connections in the Go connection pool",
	})
)

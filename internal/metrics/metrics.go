package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Metrics
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)
)

// Event Metrics
var (
	EventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total number of events published",
		},
		[]string{"type"},
	)

	EventHandlerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "event_handler_errors_total",
			Help: "Total number of event handler errors",
		},
		[]string{"type"},
	)
)

// Business Metrics
var (
	ItemsSold = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_sold_total",
			Help: "Total number of items sold",
		},
		[]string{"item"},
	)

	ItemsBought = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_bought_total",
			Help: "Total number of items bought",
		},
		[]string{"item"},
	)

	ItemsUpgraded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_upgraded_total",
			Help: "Total number of items upgraded",
		},
		[]string{"source_item", "result_item"},
	)

	ItemsDisassembled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_disassembled_total",
			Help: "Total number of items disassembled",
		},
		[]string{"item"},
	)

	ItemsUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "items_used_total",
			Help: "Total number of items used",
		},
		[]string{"item"},
	)

	SearchesPerformed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "searches_performed_total",
			Help: "Total number of searches performed",
		},
	)
)

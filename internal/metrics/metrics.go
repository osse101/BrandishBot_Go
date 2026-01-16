package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Metrics
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameHTTPRequestsTotal,
			Help: HelpTextHTTPRequestsTotal,
		},
		[]string{LabelMethod, LabelPath, LabelStatus},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    MetricNameHTTPRequestDuration,
			Help:    HelpTextHTTPRequestDuration,
			Buckets: HTTPLatencyBuckets,
		},
		[]string{LabelMethod, LabelPath},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: MetricNameHTTPRequestsInFlight,
			Help: HelpTextHTTPRequestsInFlight,
		},
	)
)

// Event Metrics
var (
	EventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameEventsPublished,
			Help: HelpTextEventsPublished,
		},
		[]string{LabelType},
	)

	EventHandlerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameEventHandlerErrors,
			Help: HelpTextEventHandlerErrors,
		},
		[]string{LabelType},
	)
)

// Business Metrics
var (
	ItemsSold = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameItemsSold,
			Help: HelpTextItemsSold,
		},
		[]string{LabelItem},
	)

	ItemsBought = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameItemsBought,
			Help: HelpTextItemsBought,
		},
		[]string{LabelItem},
	)

	ItemsUpgraded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameItemsUpgraded,
			Help: HelpTextItemsUpgraded,
		},
		[]string{LabelSourceItem, LabelResultItem},
	)

	ItemsDisassembled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameItemsDisassembled,
			Help: HelpTextItemsDisassembled,
		},
		[]string{LabelItem},
	)

	ItemsUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricNameItemsUsed,
			Help: HelpTextItemsUsed,
		},
		[]string{LabelItem},
	)

	SearchesPerformed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: MetricNameSearchesPerformed,
			Help: HelpTextSearchesPerformed,
		},
	)

	MoneyEarned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: MetricNameMoneyEarned,
			Help: HelpTextMoneyEarned,
		},
	)

	MoneySpent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: MetricNameMoneySpent,
			Help: HelpTextMoneySpent,
		},
	)
)

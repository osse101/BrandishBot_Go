package metrics

// ============================================================================
// Metric Names
// ============================================================================

// HTTP metric names
const (
	MetricNameHTTPRequestsTotal      = "http_requests_total"
	MetricNameHTTPRequestDuration    = "http_request_duration_seconds"
	MetricNameHTTPRequestsInFlight   = "http_requests_in_flight"
)

// Event metric names
const (
	MetricNameEventsPublished      = "events_published_total"
	MetricNameEventHandlerErrors   = "event_handler_errors_total"
)

// Business metric names
const (
	MetricNameItemsSold         = "items_sold_total"
	MetricNameItemsBought       = "items_bought_total"
	MetricNameItemsUpgraded     = "items_upgraded_total"
	MetricNameItemsDisassembled = "items_disassembled_total"
	MetricNameItemsUsed         = "items_used_total"
	MetricNameSearchesPerformed = "searches_performed_total"
	MetricNameMoneyEarned       = "money_earned_total"
	MetricNameMoneySpent        = "money_spent_total"
)

// ============================================================================
// Metric Help Text
// ============================================================================

// HTTP metric help text
const (
	HelpTextHTTPRequestsTotal     = "Total number of HTTP requests"
	HelpTextHTTPRequestDuration   = "HTTP request latency in seconds"
	HelpTextHTTPRequestsInFlight  = "Current number of HTTP requests being served"
)

// Event metric help text
const (
	HelpTextEventsPublished     = "Total number of events published"
	HelpTextEventHandlerErrors  = "Total number of event handler errors"
)

// Business metric help text
const (
	HelpTextItemsSold         = "Total number of items sold"
	HelpTextItemsBought       = "Total number of items bought"
	HelpTextItemsUpgraded     = "Total number of items upgraded"
	HelpTextItemsDisassembled = "Total number of items disassembled"
	HelpTextItemsUsed         = "Total number of items used"
	HelpTextSearchesPerformed = "Total number of searches performed"
	HelpTextMoneyEarned       = "Total money earned from selling items"
	HelpTextMoneySpent        = "Total money spent buying items"
)

// ============================================================================
// Metric Label Names
// ============================================================================

// Common label names used across metrics
const (
	LabelMethod      = "method"
	LabelPath        = "path"
	LabelStatus      = "status"
	LabelType        = "type"
	LabelItem        = "item"
	LabelSourceItem  = "source_item"
	LabelResultItem  = "result_item"
)

// ============================================================================
// Event Types
// ============================================================================

// Event types are now defined in internal/domain/events.go
// Import github.com/osse101/BrandishBot_Go/internal/domain to use:
//   - domain.EventTypeItemSold, domain.EventTypeItemBought
//   - domain.EventTypeItemUpgraded, domain.EventTypeItemDisassembled
//   - domain.EventTypeItemUsed, domain.EventTypeSearchPerformed
//   - domain.EventTypeEngagement

// ============================================================================
// Event Payload Field Names
// ============================================================================

// Field names used when extracting values from event payloads
const (
	PayloadFieldItemName    = "item_name"
	PayloadFieldMoneyGained = "money_gained"
	PayloadFieldSourceItem  = "source_item"
	PayloadFieldResultItem  = "result_item"
	PayloadFieldItem        = "item"
)

// ============================================================================
// Histogram Buckets
// ============================================================================

// HTTPLatencyBuckets defines the histogram buckets for HTTP request duration
// in seconds. These buckets range from 1ms to 10s to capture various latency
// patterns: fast (1-10ms), normal (10-100ms), slow (100ms-1s), very slow (1-10s)
var HTTPLatencyBuckets = []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// ============================================================================
// Log Messages
// ============================================================================

// Debug log messages
const (
	LogMsgEventPayloadNotMap    = "Event payload is not a map"
	LogMsgMetricsRecorded       = "Metrics recorded for event"
)

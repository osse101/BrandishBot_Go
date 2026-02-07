package handler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/osse101/BrandishBot_Go/internal/sse"
)

// AdminMetricsResponse contains JSON-formatted metrics for the admin dashboard
type AdminMetricsResponse struct {
	HTTP     HTTPMetrics     `json:"http"`
	Events   EventMetrics    `json:"events"`
	Business BusinessMetrics `json:"business"`
	SSE      SSEMetrics      `json:"sse"`
}

type HTTPMetrics struct {
	RequestsTotalByStatus map[string]float64 `json:"requests_total_by_status"`
	AvgLatencyMs          float64            `json:"avg_latency_ms"`
	P95LatencyMs          float64            `json:"p95_latency_ms"`
	InFlight              float64            `json:"in_flight"`
}

type EventMetrics struct {
	PublishedTotalByType map[string]float64 `json:"published_total_by_type"`
	HandlerErrorsByType  map[string]float64 `json:"handler_errors_by_type"`
}

type BusinessMetrics struct {
	ItemsSold   map[string]float64 `json:"items_sold"`
	ItemsBought map[string]float64 `json:"items_bought"`
}

type SSEMetrics struct {
	ClientCount int `json:"client_count"`
}

// AdminMetricsHandler handles admin metrics requests
type AdminMetricsHandler struct {
	sseHub *sse.Hub
}

// NewAdminMetricsHandler creates a new admin metrics handler
func NewAdminMetricsHandler(sseHub *sse.Hub) *AdminMetricsHandler {
	return &AdminMetricsHandler{sseHub: sseHub}
}

// HandleGetMetrics returns JSON-formatted metrics from Prometheus
// GET /api/v1/admin/metrics
func (h *AdminMetricsHandler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := gatherMetrics()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to gather metrics")
		return
	}

	// Add SSE client count
	metrics.SSE.ClientCount = h.sseHub.ClientCount()

	respondJSON(w, http.StatusOK, metrics)
}

func gatherMetrics() (*AdminMetricsResponse, error) {
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}

	resp := &AdminMetricsResponse{
		HTTP: HTTPMetrics{
			RequestsTotalByStatus: make(map[string]float64),
		},
		Events: EventMetrics{
			PublishedTotalByType: make(map[string]float64),
			HandlerErrorsByType:  make(map[string]float64),
		},
		Business: BusinessMetrics{
			ItemsSold:   make(map[string]float64),
			ItemsBought: make(map[string]float64),
		},
	}

	for _, mf := range metricFamilies {
		switch mf.GetName() {
		case "http_requests_total":
			for _, m := range mf.GetMetric() {
				status := getLabelValue(m, "status")
				if status != "" {
					resp.HTTP.RequestsTotalByStatus[status] += m.GetCounter().GetValue()
				}
			}
		case "http_request_duration_seconds":
			// Calculate avg and p95 from histogram
			for _, m := range mf.GetMetric() {
				hist := m.GetHistogram()
				if hist != nil {
					// Average latency
					if hist.GetSampleCount() > 0 {
						resp.HTTP.AvgLatencyMs = (hist.GetSampleSum() / float64(hist.GetSampleCount())) * 1000
					}
					// P95 approximation from buckets
					resp.HTTP.P95LatencyMs = estimateQuantile(hist, 0.95) * 1000
				}
			}
		case "http_requests_in_flight":
			for _, m := range mf.GetMetric() {
				resp.HTTP.InFlight += m.GetGauge().GetValue()
			}
		case "events_published_total":
			for _, m := range mf.GetMetric() {
				eventType := getLabelValue(m, "type")
				if eventType != "" {
					resp.Events.PublishedTotalByType[eventType] += m.GetCounter().GetValue()
				}
			}
		case "event_handler_errors_total":
			for _, m := range mf.GetMetric() {
				eventType := getLabelValue(m, "type")
				if eventType != "" {
					resp.Events.HandlerErrorsByType[eventType] += m.GetCounter().GetValue()
				}
			}
		case "items_sold_total":
			for _, m := range mf.GetMetric() {
				item := getLabelValue(m, "item")
				if item != "" {
					resp.Business.ItemsSold[item] += m.GetCounter().GetValue()
				}
			}
		case "items_bought_total":
			for _, m := range mf.GetMetric() {
				item := getLabelValue(m, "item")
				if item != "" {
					resp.Business.ItemsBought[item] += m.GetCounter().GetValue()
				}
			}
		}
	}

	return resp, nil
}

func getLabelValue(m *dto.Metric, labelName string) string {
	for _, label := range m.GetLabel() {
		if label.GetName() == labelName {
			return label.GetValue()
		}
	}
	return ""
}

// estimateQuantile approximates the given quantile from a histogram
func estimateQuantile(hist *dto.Histogram, quantile float64) float64 {
	totalCount := hist.GetSampleCount()
	if totalCount == 0 {
		return 0
	}

	targetCount := float64(totalCount) * quantile
	var cumulativeCount uint64

	buckets := hist.GetBucket()
	for _, bucket := range buckets {
		cumulativeCount = bucket.GetCumulativeCount()
		if float64(cumulativeCount) >= targetCount {
			return bucket.GetUpperBound()
		}
	}

	// If we reach here, return the last bucket's upper bound
	if len(buckets) > 0 {
		return buckets[len(buckets)-1].GetUpperBound()
	}
	return 0
}

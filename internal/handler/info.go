package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/info"
)

// InfoResponse represents the structure for info responses
type InfoResponse struct {
	Platform    string `json:"platform"`
	Feature     string `json:"feature,omitempty"`
	Topic       string `json:"topic,omitempty"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

// HandleGetInfo handles the /info endpoint
func HandleGetInfo(loader *info.Loader) http.HandlerFunc {
	formatter := info.NewFormatter()

	return func(w http.ResponseWriter, r *http.Request) {
		platform, ok := GetQueryParam(r, w, "platform")
		if !ok {
			return
		}
		feature := r.URL.Query().Get("feature")
		topic := r.URL.Query().Get("topic")

		platform = strings.ToLower(platform)
		feature = strings.ToLower(feature)
		topic = strings.ToLower(topic)

		var response InfoResponse
		response.Platform = platform

		// Handle topic request
		if feature != "" && topic != "" {
			topicData, ok := loader.GetTopic(feature, topic)
			if !ok {
				RespondError(w, http.StatusNotFound, fmt.Sprintf("Topic '%s' not found in feature '%s'", topic, feature))
				return
			}
			response.Feature = feature
			response.Topic = topic
			response.Link = ""
			response.Description = formatter.FormatTopic(topicData, platform)
			RespondJSON(w, http.StatusOK, response)
			return
		}

		// Handle feature request
		if feature != "" {
			featureData, ok := loader.GetFeature(feature)
			if !ok {
				// Feature not found - try searching topics across all features
				topicData, featureName, found := loader.SearchTopic(feature)
				if !found {
					RespondError(w, http.StatusNotFound, fmt.Sprintf("Feature or topic '%s' not found", feature))
					return
				}
				// Found as a topic in another feature
				response.Feature = featureName
				response.Topic = feature
				response.Link = ""
				response.Description = formatter.FormatTopic(topicData, platform)
				RespondJSON(w, http.StatusOK, response)
				return
			}
			response.Feature = feature
			response.Link = ""
			response.Description = formatter.FormatFeature(featureData, platform)
			RespondJSON(w, http.StatusOK, response)
			return
		}

		// Handle general info list (overview)
		overview, ok := loader.GetOverview()
		if !ok {
			RespondError(w, http.StatusNotFound, "Overview not found")
			return
		}
		response.Description = formatter.FormatFeature(overview, platform)
		RespondJSON(w, http.StatusOK, response)
	}
}

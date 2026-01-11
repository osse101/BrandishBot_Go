package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/features"
)

// InfoResponse represents the structure for info responses
type InfoResponse struct {
	Platform    string `json:"platform"`
	Feature     string `json:"feature,omitempty"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

const (
	// Placeholder for the Gist link
	gistLink = "https://gist.github.com/placeholder-gist-id"
)

// HandleGetInfo handles the /info endpoint
func HandleGetInfo(loader *features.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform := r.URL.Query().Get("platform")
		feature := r.URL.Query().Get("feature")

		if platform == "" {
			respondError(w, http.StatusBadRequest, "platform parameter is required")
			return
		}

		platform = strings.ToLower(platform)
		feature = strings.ToLower(feature)

		var response InfoResponse
		response.Platform = platform
		response.Link = gistLink

		// Base content
		if feature != "" {
			data, ok := loader.GetFeature(feature)
			if !ok {
				respondError(w, http.StatusNotFound, fmt.Sprintf("Feature '%s' not found", feature))
				return
			}
			response.Feature = feature
			// If feature is present, we generally omit the link
			response.Link = "" 
			
			// Format description based on platform
			switch platform {
			case domain.PlatformDiscord:
				var sb strings.Builder
				
				// Header and Description in blockquote
				sb.WriteString(fmt.Sprintf("# %s\n> %s", strings.ToUpper(feature), data.Description))
				if len(data.Commands) > 0 {
					sb.WriteString("\n\n**Commands**")
					for _, cmd := range data.Commands {
						sb.WriteString(fmt.Sprintf("\n• `%s`", cmd))
					}
				}
				
				response.Description = sb.String()

			case domain.PlatformTwitch, domain.PlatformYoutube:
				if len(data.Commands) > 0 {
					cmds := strings.Join(data.Commands, ", ")
					response.Description = fmt.Sprintf("[%s] Commands: %s", strings.ToUpper(feature), cmds)
				} else {
					response.Description = fmt.Sprintf("[%s] %s", strings.ToUpper(feature), data.Description)
				}

			default:
				response.Description = fmt.Sprintf("%s: %s", feature, data.Description)
			}
		} else {
			allFeatures := loader.GetAllFeatures()
			var featureList []string
			for k := range allFeatures {
				featureList = append(featureList, k)
			}

			if platform == domain.PlatformDiscord {
				response.Description = fmt.Sprintf("**BrandishBot Features**\nAvailable: %s\n\nFull documentation: %s", strings.Join(featureList, ", "), gistLink)
			} else {
				response.Description = fmt.Sprintf("BrandishBot Features: %s %s", strings.Join(featureList, ", "), gistLink)
			}
		}

		respondJSON(w, http.StatusOK, response)
	}
}

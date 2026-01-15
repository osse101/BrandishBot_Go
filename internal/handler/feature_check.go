package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

// CheckFeatureLocked checks if a feature is unlocked. If locked, it writes the appropriate error response
// and returns true (indicating "is locked"). If unlocked, it returns false.
func CheckFeatureLocked(w http.ResponseWriter, r *http.Request, svc progression.Service, key string) bool {
	log := logger.FromContext(r.Context())
	unlocked, err := svc.IsFeatureUnlocked(r.Context(), key)
	if err != nil {
		log.Error("Failed to check feature unlock status", "error", err, "feature", key)
		respondError(w, http.StatusInternalServerError, ErrMsgFeatureCheckFailed)
		return true
	}
	if !unlocked {
		log.Warn("Feature is locked", "feature", key)

		nodes, err := svc.GetRequiredNodes(r.Context(), key)
		if err != nil {
			log.Error("Failed to get required nodes", "error", err, "feature", key)
			respondError(w, http.StatusForbidden, ErrMsgFeatureLocked)
			return true
		}

		var names []string
		for _, n := range nodes {
			names = append(names, n.DisplayName)
		}

		msg := ErrMsgFeatureLocked
		if len(names) > 0 {
			msg = fmt.Sprintf("LOCKED_NODES: %s", strings.Join(names, ", "))
		}

		respondError(w, http.StatusForbidden, msg)
		return true
	}
	return false
}

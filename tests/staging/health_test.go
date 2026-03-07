//go:build staging

package staging

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	resp, _ := makeRequest(t, "GET", "/healthz", nil)

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

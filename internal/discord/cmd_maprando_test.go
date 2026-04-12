package discord

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMapRandoClient checks the MapRandoClient logic with a mock HTTP server
func TestMapRandoClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/randomize", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		err := r.ParseMultipartForm(10 << 20)
		assert.NoError(t, err)
		assert.Equal(t, "secret-token", r.FormValue("spoiler_token"))
		assert.Equal(t, `{"skills": true}`, r.FormValue("settings"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"seed_url": "/seed/my-test-seed"}`)
	})
	mux.HandleFunc("/seed/my-test-seed/unlock", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "secret-token", r.FormValue("spoiler_token"))
		// Simulate redirect on success
		http.Redirect(w, r, "/seed/my-test-seed", http.StatusFound)
	})
	mux.HandleFunc("/seed/my-test-seed", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Seed page!")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewMapRandoClient(server.URL, "secret-token")
	// Inject a fake preset
	client.presetNames = []string{"test-preset"}
	client.presets["test-preset"] = `{"skills": true}`

	// Test Randomize
	seedURL, err := client.Randomize("test-preset", nil)
	assert.NoError(t, err)
	assert.Equal(t, server.URL+"/seed/my-test-seed", seedURL)

	// Test Unlock
	err = client.Unlock("my-test-seed", "test-preset")
	assert.NoError(t, err)

	// Test devonly domain switching
	client.baseURL = "https://maprando.com"
	client.devOnly["test-preset"] = true
	assert.Equal(t, "https://dev.maprando.com/seed/my-seed", client.SeedURL("my-seed", "test-preset"))
	client.devOnly["test-preset"] = false
	assert.Equal(t, "https://maprando.com/seed/my-seed", client.SeedURL("my-seed", "test-preset"))

	// Test unknown preset
	_, err = client.Randomize("unknown", nil)
	assert.Error(t, err)
}

// TestMapRandoCommandFactories ensures the command factories return well-formed commands
func TestMapRandoCommandFactories(t *testing.T) {
	client := NewMapRandoClient("http://localhost:9000", "token")

	cmd, handler := MapRandoCommand(client)
	assert.NotNil(t, handler)
	assert.Equal(t, "maprando", cmd.Name)
	assert.Equal(t, 10, len(cmd.Options))
	assert.True(t, cmd.Options[0].Autocomplete)

	cmdUnlock, handlerUnlock := MapRandoUnlockCommand(client)
	assert.NotNil(t, handlerUnlock)
	assert.Equal(t, "maprandounlock", cmdUnlock.Name)
	assert.Equal(t, 1, len(cmdUnlock.Options))
	assert.Equal(t, "seed", cmdUnlock.Options[0].Name)
}

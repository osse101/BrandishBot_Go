package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestJobFlow_Integration(t *testing.T) {
	// 1. Setup mock server
	mux := http.NewServeMux()

	// Mock RegisterUser
	mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		user := domain.User{
			ID:       "user-123",
			Username: "Tester",
			TwitchID: "12345",
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	})

	// Mock AdminAwardXP
	mux.HandleFunc("/api/v1/admin/job/award-xp", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var resp struct {
			Success bool `json:"success"`
			Result  struct {
				LeveledUp bool `json:"leveled_up"`
				NewLevel  int  `json:"new_level"`
				NewXP     int  `json:"new_xp"`
			} `json:"result"`
		}
		resp.Success = true
		resp.Result.LeveledUp = true
		resp.Result.NewLevel = 2
		resp.Result.NewXP = 100

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// Mock GetUserJobs
	mux.HandleFunc("/api/v1/jobs/user", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, domain.PlatformDiscord, r.URL.Query().Get("platform"))
		assert.Equal(t, "12345", r.URL.Query().Get("platform_id"))

		resp := UserJobsResponse{
			Platform:   domain.PlatformDiscord,
			PlatformID: "12345",
			PrimaryJob: &domain.UserJobInfo{
				JobKey:      domain.JobKeyBlacksmith,
				DisplayName: "Blacksmith",
				Level:       2,
				CurrentXP:   100,
			},
			Jobs: []domain.UserJobInfo{
				{
					JobKey:      domain.JobKeyBlacksmith,
					DisplayName: "Blacksmith",
					Level:       2,
					CurrentXP:   100,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// 2. Initialize client
	client := NewAPIClient(server.URL, "test-api-key")

	// 3. Execution flow

	// A. Create/Register user
	user, err := client.RegisterUser("Tester", "12345")
	assert.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)

	// B. Award XP
	awardResult, err := client.AdminAwardXP(domain.PlatformDiscord, "Tester", domain.JobKeyBlacksmith, 100)
	assert.NoError(t, err)
	assert.True(t, awardResult.LeveledUp)
	assert.Equal(t, 2, awardResult.NewLevel)

	// C. Fetch Job Status
	jobsResp, err := client.GetUserJobs(domain.PlatformDiscord, "12345")
	assert.NoError(t, err)
	assert.NotNil(t, jobsResp.PrimaryJob)
	assert.Equal(t, domain.JobKeyBlacksmith, jobsResp.PrimaryJob.JobKey)
	assert.Equal(t, 2, jobsResp.PrimaryJob.Level)
	assert.Len(t, jobsResp.Jobs, 1)
	assert.Equal(t, domain.JobKeyBlacksmith, jobsResp.Jobs[0].JobKey)
}

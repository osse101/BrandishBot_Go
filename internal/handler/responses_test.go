package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestRespondJSON(t *testing.T) {
	t.Run("writes JSON payload and correct status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		payload := DataResponse{
			Message: "Success!",
			Data:    map[string]string{"key": "value"},
		}

		RespondJSON(rec, http.StatusOK, payload)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp DataResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "Success!", resp.Message)
		// type assertion since it unmarshals interface{} back to map[string]interface{}
		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", dataMap["key"])
	})
}

func TestRespondError(t *testing.T) {
	t.Run("writes JSON error payload and correct status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errMsg := "Internal Server Error"

		RespondError(rec, http.StatusInternalServerError, errMsg)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, errMsg, resp.Error)
	})
}

func TestMapServiceErrorToUserMessage(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedMsg  string
	}{
		// nil case
		{
			name:         "nil error",
			err:          nil,
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  ErrMsgUnknownError,
		},

		// User Errors
		{
			name:         "ErrUserNotFound",
			err:          domain.ErrUserNotFound,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgUserNotFoundError,
		},
		{
			name:         "ErrInvalidPlatform",
			err:          domain.ErrInvalidPlatform,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgInvalidPlatformError,
		},

		// Item/Inventory Errors
		{
			name:         "ErrItemNotFound",
			err:          domain.ErrItemNotFound,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgItemNotFoundError,
		},
		{
			name:         "ErrInsufficientFunds",
			err:          domain.ErrInsufficientFunds,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotEnoughMoneyError,
		},
		{
			name:         "ErrInsufficientQuantity",
			err:          domain.ErrInsufficientQuantity,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgInsufficientItemsErr,
		},
		{
			name:         "ErrNotInInventory",
			err:          domain.ErrNotInInventory,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotInInventoryError,
		},
		{
			name:         "ErrInventoryFull",
			err:          domain.ErrInventoryFull,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgInventoryFullError,
		},
		{
			name:         "ErrNotSellable",
			err:          domain.ErrNotSellable,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotSellableError,
		},
		{
			name:         "ErrNotBuyable",
			err:          domain.ErrNotBuyable,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotBuyableError,
		},

		// Compost Errors
		{
			name:         "ErrCompostBinFull",
			err:          domain.ErrCompostBinFull,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgCompostBinFull,
		},
		{
			name:         "ErrCompostNotCompostable",
			err:          domain.ErrCompostNotCompostable,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgCompostNotCompostable,
		},
		{
			name:         "ErrCompostMustHarvest",
			err:          domain.ErrCompostMustHarvest,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgCompostMustHarvest,
		},
		{
			name:         "ErrCompostNothingToHarvest",
			err:          domain.ErrCompostNothingToHarvest,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgCompostNothingToHarvest,
		},

		// Economy & Feature Errors
		{
			name:         "ErrRecipeLocked",
			err:          domain.ErrRecipeLocked,
			expectedCode: http.StatusForbidden,
			expectedMsg:  ErrMsgRecipeLockedError,
		},
		{
			name:         "ErrFeatureLocked",
			err:          domain.ErrFeatureLocked,
			expectedCode: http.StatusForbidden,
			expectedMsg:  ErrMsgFeatureLockedProgressionError,
		},
		{
			name:         "ErrDailyCapReached",
			err:          domain.ErrDailyCapReached,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgDailyCapReachedError,
		},
		{
			name:         "ErrOnCooldown (generic)",
			err:          domain.ErrOnCooldown,
			expectedCode: http.StatusTooManyRequests,
			expectedMsg:  ErrMsgOnCooldownError,
		},
		{
			name:         "ErrOnCooldown (typed)",
			err:          cooldown.ErrOnCooldown{Remaining: 5 * time.Second},
			expectedCode: http.StatusTooManyRequests,
			expectedMsg:  "You can  again in 5s",
		},
		{
			name:         "ErrRecipeNotFound",
			err:          domain.ErrRecipeNotFound,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgRecipeNotFoundError,
		},
		{
			name:         "ErrUserAlreadyVoted",
			err:          domain.ErrUserAlreadyVoted,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgAlreadyVotedError,
		},

		// Gamble Errors
		{
			name:         "ErrGambleNotFound",
			err:          domain.ErrGambleNotFound,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgGambleNotFoundError,
		},
		{
			name:         "ErrGambleAlreadyActive",
			err:          domain.ErrGambleAlreadyActive,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgGambleAlreadyActiveError,
		},
		{
			name:         "ErrNotInJoiningState",
			err:          domain.ErrNotInJoiningState,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotAcceptingParticipantsErr,
		},
		{
			name:         "ErrJoinDeadlinePassed",
			err:          domain.ErrJoinDeadlinePassed,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgJoinDeadlinePassedError,
		},
		{
			name:         "ErrAtLeastOneLootboxRequired",
			err:          domain.ErrAtLeastOneLootboxRequired,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgLootboxRequiredError,
		},
		{
			name:         "ErrBetQuantityMustBePositive",
			err:          domain.ErrBetQuantityMustBePositive,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgBetQuantityPositiveError,
		},
		{
			name:         "ErrNotALootbox",
			err:          domain.ErrNotALootbox,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgNotLootboxError,
		},
		{
			name:         "ErrUserAlreadyJoined",
			err:          domain.ErrUserAlreadyJoined,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  ErrMsgAlreadyJoinedError,
		},

		// System Errors
		{
			name:         "ErrDatabaseError",
			err:          domain.ErrDatabaseError,
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  ErrMsgGenericServerError,
		},
		{
			name:         "ErrConnectionTimeout",
			err:          domain.ErrConnectionTimeout,
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  ErrMsgGenericServerError,
		},
		{
			name:         "ErrDeadlockDetected",
			err:          domain.ErrDeadlockDetected,
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  ErrMsgGenericServerError,
		},

		// Fallback specific message for tests/custom errors
		{
			name:         "custom error string",
			err:          errors.New("a short specific error for tests"),
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  "a short specific error for tests",
		},
		{
			name:         "long error string",
			err:          errors.New("this is a very long error message that exceeds the internal threshold of two hundred characters so it should be truncated or default to a generic error message, avoiding exposing internal system details to users in production environment"),
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := MapServiceErrorToUserMessage(tt.err)
			assert.Equal(t, tt.expectedCode, code, "expected HTTP status code %d but got %d", tt.expectedCode, code)
			assert.Equal(t, tt.expectedMsg, msg, "expected message %q but got %q", tt.expectedMsg, msg)
		})
	}
}

func TestMapServiceErrorToUserMessage_Wrapped(t *testing.T) {
	t.Run("unwraps errors correctly", func(t *testing.T) {
		err := fmt.Errorf("some extra context: %w", domain.ErrUserNotFound)
		code, msg := MapServiceErrorToUserMessage(err)
		assert.Equal(t, http.StatusBadRequest, code)
		// ErrUserNotFound has specific string matching handling, where it will just output the errMsg as a specific message for users
		assert.Equal(t, "some extra context: user not found", msg)
	})

	t.Run("unwraps deeply nested errors", func(t *testing.T) {
		err := fmt.Errorf("layer 1: %w", fmt.Errorf("layer 2: %w", domain.ErrFeatureLocked))
		code, msg := MapServiceErrorToUserMessage(err)
		assert.Equal(t, http.StatusForbidden, code)
		assert.Equal(t, ErrMsgFeatureLockedProgressionError, msg)
	})

	t.Run("handles specific error formatting for recipe not found", func(t *testing.T) {
		err := fmt.Errorf("recipe for diamond_sword %w", domain.ErrRecipeNotFound)
		code, _ := MapServiceErrorToUserMessage(err)
		assert.Equal(t, http.StatusBadRequest, code)
		// Our custom error mapper has a cutSuffix check for recipe errors
		// It checks if it ends with " | recipe not found"
		err2 := fmt.Errorf("diamond_sword | %w", domain.ErrRecipeNotFound)
		code2, msg2 := MapServiceErrorToUserMessage(err2)
		assert.Equal(t, http.StatusBadRequest, code2)
		assert.Equal(t, "diamond_sword", msg2)
	})
}

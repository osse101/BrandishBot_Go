package discord

import (
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// LinkInitiateResult is the response from InitiateLink
type LinkInitiateResult struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// LinkClaimResult is the response from ClaimLink
type LinkClaimResult struct {
	SourcePlatform       string `json:"source_platform"`
	AwaitingConfirmation bool   `json:"awaiting_confirmation"`
}

// LinkConfirmResult is the response from ConfirmLink
type LinkConfirmResult struct {
	Success         bool     `json:"success"`
	LinkedPlatforms []string `json:"linked_platforms"`
}

// InitiateLink initiates a cross-platform link (Step 1)
func (c *APIClient) InitiateLink(discordID string) (*LinkInitiateResult, error) {
	req := map[string]string{
		"platform":    domain.PlatformDiscord,
		"platform_id": discordID,
	}

	var result LinkInitiateResult
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/link/initiate", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ClaimLink claims a link token (Step 2)
func (c *APIClient) ClaimLink(token, discordID string) (*LinkClaimResult, error) {
	req := map[string]string{
		"token":       token,
		"platform":    domain.PlatformDiscord,
		"platform_id": discordID,
	}

	var result LinkClaimResult
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/link/claim", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConfirmLink confirms a pending link (Step 3)
func (c *APIClient) ConfirmLink(discordID string) (*LinkConfirmResult, error) {
	req := map[string]string{
		"platform":    domain.PlatformDiscord,
		"platform_id": discordID,
	}

	var result LinkConfirmResult
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/link/confirm", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// InitiateUnlink starts the unlink process
func (c *APIClient) InitiateUnlink(discordID, targetPlatform string) error {
	req := map[string]interface{}{
		"platform":        domain.PlatformDiscord,
		"platform_id":     discordID,
		"target_platform": targetPlatform,
		"confirm":         false,
	}

	return c.doRequestAndParse(http.MethodPost, "/api/v1/link/unlink", req, nil)
}

// ConfirmUnlink confirms the unlink
func (c *APIClient) ConfirmUnlink(discordID, targetPlatform string) error {
	req := map[string]interface{}{
		"platform":        domain.PlatformDiscord,
		"platform_id":     discordID,
		"target_platform": targetPlatform,
		"confirm":         true,
	}

	return c.doRequestAndParse(http.MethodPost, "/api/v1/link/unlink", req, nil)
}

// GetLinkStatus gets current link status
func (c *APIClient) GetLinkStatus(discordID string) ([]string, error) {
	path := fmt.Sprintf("/api/v1/link/status?platform=%s&platform_id=%s", domain.PlatformDiscord, discordID)
	var result struct {
		LinkedPlatforms []string `json:"linked_platforms"`
	}
	if err := c.doRequestAndParse(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.LinkedPlatforms, nil
}

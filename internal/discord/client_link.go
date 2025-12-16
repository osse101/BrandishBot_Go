package discord

import (
	"encoding/json"
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

	resp, err := c.doRequest(http.MethodPost, "/link/initiate", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result LinkInitiateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

	resp, err := c.doRequest(http.MethodPost, "/link/claim", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result LinkClaimResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ConfirmLink confirms a pending link (Step 3)
func (c *APIClient) ConfirmLink(discordID string) (*LinkConfirmResult, error) {
	req := map[string]string{
		"platform":    domain.PlatformDiscord,
		"platform_id": discordID,
	}

	resp, err := c.doRequest(http.MethodPost, "/link/confirm", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result LinkConfirmResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

	resp, err := c.doRequest(http.MethodPost, "/link/unlink", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	return nil
}

// ConfirmUnlink confirms the unlink
func (c *APIClient) ConfirmUnlink(discordID, targetPlatform string) error {
	req := map[string]interface{}{
		"platform":        domain.PlatformDiscord,
		"platform_id":     discordID,
		"target_platform": targetPlatform,
		"confirm":         true,
	}

	resp, err := c.doRequest(http.MethodPost, "/link/unlink", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	return nil
}

// GetLinkStatus gets current link status
func (c *APIClient) GetLinkStatus(discordID string) ([]string, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/link/status?platform=%s&platform_id=%s", domain.PlatformDiscord, discordID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		LinkedPlatforms []string `json:"linked_platforms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.LinkedPlatforms, nil
}

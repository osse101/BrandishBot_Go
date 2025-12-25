package progression

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Notifier handles progression notifications to external services
type Notifier struct {
	discordWebhookURL     string
	streamerbotWebhookURL string
	httpClient            *http.Client
}

// NewNotifier creates a new progression notifier
func NewNotifier(discordWebhookURL, streamerbotWebhookURL string) *Notifier {
	return &Notifier{
		discordWebhookURL:     discordWebhookURL,
		streamerbotWebhookURL: streamerbotWebhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Subscribe registers the notifier to listen for progression events
func (n *Notifier) Subscribe(bus event.Bus) {
	bus.Subscribe(event.ProgressionCycleCompleted, n.handleProgressionCycleCompleted)
}

// handleProgressionCycleCompleted processes progression cycle completion events
func (n *Notifier) handleProgressionCycleCompleted(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		log.Warn("Invalid payload type for progression cycle completed event")
		return nil
	}

	unlockedNode, ok := payload["unlocked_node"].(*domain.ProgressionNode)
	if !ok {
		log.Warn("Missing or invalid unlocked_node in event payload")
		return nil
	}

	votingSession, ok := payload["voting_session"].(*domain.ProgressionVotingSession)
	if !ok {
		log.Warn("Missing or invalid voting_session in event payload")
		return nil
	}

	// Send to Discord
	if n.discordWebhookURL != "" {
		if err := n.sendDiscordNotification(ctx, unlockedNode, votingSession); err != nil {
			log.Error("Failed to send Discord notification", "error", err)
			// Don't fail the event handler, just log it
		}
	}

	// Send to Streamer.bot
	if n.streamerbotWebhookURL != "" {
		if err := n.sendStreamerbotNotification(ctx, unlockedNode, votingSession); err != nil {
			log.Error("Failed to send Streamer.bot notification", "error", err)
			// Don't fail the event handler, just log it
		}
	}

	return nil
}

// sendDiscordNotification sends a formatted announcement to Discord
func (n *Notifier) sendDiscordNotification(ctx context.Context, unlockedNode *domain.ProgressionNode, votingSession *domain.ProgressionVotingSession) error {
	log := logger.FromContext(ctx)

	// Format voting options
	var optionsText strings.Builder
	for i, option := range votingSession.Options {
		if option.NodeDetails != nil {
			optionsText.WriteString(fmt.Sprintf("\n%d. **%s** - %s",
				i+1,
				option.NodeDetails.DisplayName,
				option.NodeDetails.Description))
		}
	}

	description := fmt.Sprintf("🎉 **%s** has been unlocked!\n\n📊 **Vote for the next unlock:**%s\n\nUse `/vote <node_key>` to cast your vote!",
		unlockedNode.DisplayName,
		optionsText.String())

	payload := map[string]interface{}{
		"title":       "🏆 Progression Milestone Reached!",
		"description": description,
		"color":       0x00FF00, // Green
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.discordWebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord webhook returned status: %d", resp.StatusCode)
	}

	log.Info("Sent Discord notification", "node", unlockedNode.NodeKey)
	return nil
}

// sendStreamerbotNotification sends a notification to Streamer.bot
func (n *Notifier) sendStreamerbotNotification(ctx context.Context, unlockedNode *domain.ProgressionNode, votingSession *domain.ProgressionVotingSession) error {
	log := logger.FromContext(ctx)

	// Format options for Streamer.bot
	options := make([]map[string]interface{}, 0, len(votingSession.Options))
	for _, option := range votingSession.Options {
		if option.NodeDetails != nil {
			options = append(options, map[string]interface{}{
				"node_key":     option.NodeDetails.NodeKey,
				"display_name": option.NodeDetails.DisplayName,
				"description":  option.NodeDetails.Description,
				"vote_count":   option.VoteCount,
			})
		}
	}

	payload := map[string]interface{}{
		"event": "progression_unlock",
		"data": map[string]interface{}{
			"unlocked_node": map[string]interface{}{
				"node_key":     unlockedNode.NodeKey,
				"display_name": unlockedNode.DisplayName,
				"description":  unlockedNode.Description,
			},
			"voting_session": map[string]interface{}{
				"session_id": votingSession.ID,
				"options":    options,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Streamer.bot payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.streamerbotWebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Streamer.bot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Streamer.bot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Streamer.bot webhook returned status: %d", resp.StatusCode)
	}

	log.Info("Sent Streamer.bot notification", "node", unlockedNode.NodeKey)
	return nil
}

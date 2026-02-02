package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SSENotifier handles sending Discord notifications for SSE events
type SSENotifier struct {
	session            *discordgo.Session
	notificationChanID string
}

// NewSSENotifier creates a new SSE notifier
func NewSSENotifier(session *discordgo.Session, notificationChanID string) *SSENotifier {
	return &SSENotifier{
		session:            session,
		notificationChanID: notificationChanID,
	}
}

// RegisterHandlers registers all SSE event handlers with the client
func (n *SSENotifier) RegisterHandlers(client *SSEClient) {
	client.OnEvent(SSEEventTypeJobLevelUp, n.handleJobLevelUp)
	client.OnEvent(SSEEventTypeVotingStarted, n.handleVotingStarted)
	client.OnEvent(SSEEventTypeCycleCompleted, n.handleCycleCompleted)
	client.OnEvent(SSEEventTypeAllUnlocked, n.handleAllUnlocked)
	client.OnEvent(SSEEventTypeGambleCompleted, n.handleGambleCompleted)
}

// JobLevelUpPayload is the payload for job level up events
type JobLevelUpPayload struct {
	UserID   string `json:"user_id"`
	JobKey   string `json:"job_key"`
	OldLevel int    `json:"old_level"`
	NewLevel int    `json:"new_level"`
	Source   string `json:"source,omitempty"`
}

// VotingStartedPayload is the payload for voting started events
type VotingStartedPayload struct {
	SessionID      int                `json:"session_id"`
	NodeKey        string             `json:"node_key,omitempty"`
	TargetLevel    int                `json:"target_level"`
	AutoSelected   bool               `json:"auto_selected"`
	Options        []VotingOptionInfo `json:"options,omitempty"`
	PreviousUnlock string             `json:"previous_unlock"`
}

// VotingOptionInfo contains voting option details
type VotingOptionInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
}

// CycleCompletedPayload is the payload for cycle completed events
type CycleCompletedPayload struct {
	UnlockedNode  NodeInfo           `json:"unlocked_node"`
	VotingSession *VotingSessionInfo `json:"voting_session,omitempty"`
}

// NodeInfo contains node details
type NodeInfo struct {
	NodeKey     string `json:"node_key"`
	DisplayName string `json:"display_name"`
}

// VotingSessionInfo contains voting session details
type VotingSessionInfo struct {
	SessionID int                `json:"session_id"`
	Options   []VotingOptionInfo `json:"options"`
}

// AllUnlockedPayload is the payload for all unlocked events
type AllUnlockedPayload struct {
	Message string `json:"message"`
}

// GambleCompletedPayload is the payload for gamble completed events
type GambleCompletedPayload struct {
	GambleID         string `json:"gamble_id"`
	WinnerID         string `json:"winner_id"`
	TotalValue       int64  `json:"total_value"`
	ParticipantCount int    `json:"participant_count"`
}

func (n *SSENotifier) handleJobLevelUp(event SSEEvent) error {
	if n.notificationChanID == "" {
		return nil // No notification channel configured
	}

	var payload JobLevelUpPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "event_type", event.Type)
		return nil
	}

	// Format job name nicely
	jobName := formatJobName(payload.JobKey)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Level Up! %s", jobName),
		Description: fmt.Sprintf("A user has reached **level %d** in %s!", payload.NewLevel, jobName),
		Color:       0xFFD700, // Gold
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Job",
				Value:  jobName,
				Inline: true,
			},
			{
				Name:   "New Level",
				Value:  fmt.Sprintf("%d", payload.NewLevel),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Job System",
		},
	}

	if payload.Source != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "From",
			Value:  formatSource(payload.Source),
			Inline: true,
		})
	}

	_, err := n.session.ChannelMessageSendEmbed(n.notificationChanID, embed)
	if err != nil {
		slog.Error(sseLogMsgNotificationError, "error", err, "event_type", event.Type)
		return err
	}

	slog.Info(sseLogMsgNotificationSent, "event_type", event.Type, "job", payload.JobKey, "level", payload.NewLevel)
	return nil
}

func (n *SSENotifier) handleVotingStarted(event SSEEvent) error {
	if n.notificationChanID == "" {
		return nil
	}

	var payload VotingStartedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "event_type", event.Type)
		return nil
	}

	// Build options list
	var optionsList strings.Builder
	for i, opt := range payload.Options {
		name := opt.DisplayName
		if name == "" {
			name = formatNodeKey(opt.NodeKey)
		}
		optionsList.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, name))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "New Voting Session!",
		Description: "A new progression voting session has started! Use `/vote` to cast your vote.",
		Color:       0x5865F2, // Discord Blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Options",
				Value:  optionsList.String(),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Progression System",
		},
	}

	if payload.AutoSelected {
		embed.Title = "Target Auto-Selected"
		embed.Description = fmt.Sprintf("Only one option was available. **%s** has been automatically selected as the next unlock target.", formatNodeKey(payload.NodeKey))
		embed.Fields = nil // Clear options field
	}

	_, err := n.session.ChannelMessageSendEmbed(n.notificationChanID, embed)
	if err != nil {
		slog.Error(sseLogMsgNotificationError, "error", err, "event_type", event.Type)
		return err
	}

	slog.Info(sseLogMsgNotificationSent, "event_type", event.Type, "session_id", payload.SessionID)
	return nil
}

func (n *SSENotifier) handleCycleCompleted(event SSEEvent) error {
	if n.notificationChanID == "" {
		return nil
	}

	var payload CycleCompletedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "event_type", event.Type)
		return nil
	}

	nodeName := payload.UnlockedNode.DisplayName
	if nodeName == "" {
		nodeName = formatNodeKey(payload.UnlockedNode.NodeKey)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Feature Unlocked!",
		Description: fmt.Sprintf("**%s** has been unlocked!", nodeName),
		Color:       0x57F287, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Progression System",
		},
	}

	// Add voting session info if available
	if payload.VotingSession != nil && len(payload.VotingSession.Options) > 0 {
		var optionsList strings.Builder
		for i, opt := range payload.VotingSession.Options {
			name := opt.DisplayName
			if name == "" {
				name = formatNodeKey(opt.NodeKey)
			}
			optionsList.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, name))
		}

		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Next Voting Options",
				Value:  optionsList.String(),
				Inline: false,
			},
		}
		embed.Description += "\n\nA new voting session has started! Use `/vote` to choose the next feature."
	}

	_, err := n.session.ChannelMessageSendEmbed(n.notificationChanID, embed)
	if err != nil {
		slog.Error(sseLogMsgNotificationError, "error", err, "event_type", event.Type)
		return err
	}

	slog.Info(sseLogMsgNotificationSent, "event_type", event.Type, "unlocked_node", payload.UnlockedNode.NodeKey)
	return nil
}

func (n *SSENotifier) handleAllUnlocked(event SSEEvent) error {
	if n.notificationChanID == "" {
		return nil
	}

	var payload AllUnlockedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "event_type", event.Type)
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎉 All Features Unlocked!",
		Description: payload.Message,
		Color:       0xFFD700, // Gold
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Progression System",
		},
	}

	if embed.Description == "" {
		embed.Description = "Congratulations! Every single feature and upgrade in BrandishBot has been unlocked by the community!"
	}

	_, err := n.session.ChannelMessageSendEmbed(n.notificationChanID, embed)
	if err != nil {
		slog.Error(sseLogMsgNotificationError, "error", err, "event_type", event.Type)
		return err
	}

	slog.Info(sseLogMsgNotificationSent, "event_type", event.Type)
	return nil
}

func (n *SSENotifier) handleGambleCompleted(event SSEEvent) error {
	if n.notificationChanID == "" {
		return nil
	}

	var payload GambleCompletedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "event_type", event.Type)
		return nil
	}

	title := "Gamble Completed!"
	description := ""
	color := 0x9B59B6 // Purple

	if payload.WinnerID != "" {
		description = fmt.Sprintf("The gamble has concluded! **%s** won a total value of **%d** credits from **%d** participants!",
			payload.WinnerID, payload.TotalValue, payload.ParticipantCount)
	} else {
		title = "Gamble Ended (No Winner)"
		description = fmt.Sprintf("The gamble has concluded with no winner. Total value was **%d** credits from **%d** participants.",
			payload.TotalValue, payload.ParticipantCount)
		color = 0x95A5A6 // Grey
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Gamble System",
		},
	}

	_, err := n.session.ChannelMessageSendEmbed(n.notificationChanID, embed)
	if err != nil {
		slog.Error(sseLogMsgNotificationError, "error", err, "event_type", event.Type)
		return err
	}

	slog.Info(sseLogMsgNotificationSent, "event_type", event.Type, "gamble_id", payload.GambleID)
	return nil
}

// Helper functions

func formatJobName(jobKey string) string {
	// Convert job_key to Job Key format
	parts := strings.Split(jobKey, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func formatNodeKey(nodeKey string) string {
	// Convert feature_some_feature to Some Feature
	parts := strings.Split(nodeKey, "_")
	// Skip "feature_" or "upgrade_" prefix
	if len(parts) > 1 && (parts[0] == "feature" || parts[0] == "upgrade") {
		parts = parts[1:]
	}
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func formatSource(source string) string {
	// Convert source_name to Source Name format
	parts := strings.Split(source, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

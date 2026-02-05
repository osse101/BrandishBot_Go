using System;
using System.Collections.Generic;
using Newtonsoft.Json;

namespace BrandishBot.Client
{
    // --- Core Response Types ---

    public class SuccessResponse
    {
        [JsonProperty("message")]
        public string Message { get; set; }
    }

    public class ErrorResponse
    {
        [JsonProperty("error")]
        public string Error { get; set; }

        [JsonProperty("fields")]
        public Dictionary<string, string> Fields { get; set; }
    }

    // --- Domain Models ---

    public class Item
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("name")]
        public string Name { get; set; }

        [JsonProperty("public_name")]
        public string PublicName { get; set; }

        [JsonProperty("type")]
        public string Type { get; set; }

        [JsonProperty("base_value")]
        public int BaseValue { get; set; }

        [JsonProperty("quantity")]
        public int Quantity { get; set; }
    }

    public class User
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("twitch_id")]
        public string TwitchId { get; set; }

        [JsonProperty("youtube_id")]
        public string YoutubeId { get; set; }

        [JsonProperty("discord_id")]
        public string DiscordId { get; set; }
    }

    // --- Inventory Models ---

    public class GetInventoryResponse
    {
        [JsonProperty("items")]
        public List<Item> Items { get; set; }

        [JsonProperty("user")]
        public User User { get; set; }
    }

    public class AddItemByUsernameRequest
    {
        [JsonProperty("platform")]
        public string Platform { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("item_name")]
        public string ItemName { get; set; }

        [JsonProperty("quantity")]
        public int Quantity { get; set; }
    }

    // --- Message Models ---

    public class MessageResult
    {
        [JsonProperty("user")]
        public User User { get; set; }

        [JsonProperty("found_strings")]
        public List<FoundString> FoundStrings { get; set; }
    }

    public class FoundString
    {
        [JsonProperty("trigger")]
        public string Trigger { get; set; }

        [JsonProperty("match")]
        public string Match { get; set; }
    }

    // --- Progression Models ---
    
    public class ProgressionStatus
    {
        [JsonProperty("total_nodes")]
        public int TotalNodes { get; set; }

        [JsonProperty("total_unlocked")]
        public int TotalUnlocked { get; set; }

        [JsonProperty("active_session")]
        public VotingSession ActiveSession { get; set; }
        
        [JsonProperty("active_unlock_progress")]
        public UnlockProgress ActiveUnlockProgress { get; set; }
    }

    public class VotingSession
    {
        [JsonProperty("session_id")]
        public string SessionId { get; set; }

        [JsonProperty("started_at")]
        public DateTime StartedAt { get; set; }

        [JsonProperty("options")]
        public List<VotingOption> Options { get; set; }
    }

    public class VotingOption
    {
        [JsonProperty("node_key")]
        public string NodeKey { get; set; }

        [JsonProperty("target_level")]
        public int TargetLevel { get; set; }

        [JsonProperty("vote_count")]
        public int VoteCount { get; set; }
        
        [JsonProperty("node_details")]
        public NodeDetails NodeDetails { get; set; }
    }

    public class NodeDetails
    {
        [JsonProperty("node_key")]
        public string NodeKey { get; set; }
        
        [JsonProperty("display_name")]
        public string DisplayName { get; set; }
        
        [JsonProperty("size")]
        public string Size { get; set; }
    }

    public class UnlockProgress
    {
        [JsonProperty("node_id")]
        public string NodeId { get; set; }

        [JsonProperty("node_key")]
        public string NodeKey { get; set; }

        [JsonProperty("target_node_name")]
        public string TargetNodeName { get; set; }

        [JsonProperty("contributions_accumulated")]
        public int ContributionsAccumulated { get; set; }

        [JsonProperty("target_unlock_cost")]
        public int TargetUnlockCost { get; set; }

        [JsonProperty("completion_percentage")]
        public double CompletionPercentage { get; set; }

        [JsonProperty("started_at")]
        public DateTime StartedAt { get; set; }
    }

    // --- Stats & Leaderboard Models ---

    public class StatsSummary
    {
        [JsonProperty("period")]
        public string Period { get; set; }

        [JsonProperty("start_time")]
        public DateTime StartTime { get; set; }

        [JsonProperty("end_time")]
        public DateTime EndTime { get; set; }

        [JsonProperty("total_events")]
        public int TotalEvents { get; set; }

        [JsonProperty("event_counts")]
        public Dictionary<string, int> EventCounts { get; set; }

        [JsonProperty("metrics")]
        public Dictionary<string, object> Metrics { get; set; }
    }

    public class LeaderboardEntry
    {
        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("count")]
        public int Count { get; set; }

        [JsonProperty("event_type")]
        public string EventType { get; set; }
    }

    // --- Prediction Models ---

    public class PredictionWinner
    {
        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("platform_id")]
        public string PlatformId { get; set; }

        [JsonProperty("points_won")]
        public int PointsWon { get; set; }
    }

    public class PredictionParticipant
    {
        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("platform_id")]
        public string PlatformId { get; set; }

        [JsonProperty("points_spent")]
        public int PointsSpent { get; set; }
    }

    public class PredictionResult
    {
        [JsonProperty("total_points")]
        public int TotalPoints { get; set; }

        [JsonProperty("contribution_awarded")]
        public int ContributionAwarded { get; set; }

        [JsonProperty("participants_processed")]
        public int ParticipantsProcessed { get; set; }

        [JsonProperty("winner_xp_awarded")]
        public int WinnerXpAwarded { get; set; }

        [JsonProperty("message")]
        public string Message { get; set; }
    }

    // --- Gamble Models ---

    public class StartGambleResponse
    {
        [JsonProperty("message")]
        public string Message { get; set; }

        [JsonProperty("gamble_id")]
        public string GambleId { get; set; }
    }

    public class Gamble
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("initiator_id")]
        public string InitiatorId { get; set; }

        [JsonProperty("state")]
        public string State { get; set; }

        [JsonProperty("created_at")]
        public DateTime CreatedAt { get; set; }

        [JsonProperty("join_deadline")]
        public DateTime JoinDeadline { get; set; }

        [JsonProperty("participants")]
        public List<GambleParticipant> Participants { get; set; }

        [JsonProperty("winner_id")]
        public string WinnerId { get; set; }

        [JsonProperty("total_value")]
        public long TotalValue { get; set; }
    }

    public class GambleParticipant
    {
        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("lootbox_bets")]
        public List<LootboxBet> LootboxBets { get; set; }
    }

    public class LootboxBet
    {
        [JsonProperty("item_name")]
        public string ItemName { get; set; }

        [JsonProperty("quantity")]
        public int Quantity { get; set; }
    }

    public class GambleResult
    {
        [JsonProperty("gamble_id")]
        public string GambleId { get; set; }

        [JsonProperty("winner_id")]
        public string WinnerId { get; set; }

        [JsonProperty("total_value")]
        public long TotalValue { get; set; }

        [JsonProperty("items")]
        public List<GambleOpenedItem> Items { get; set; }
    }

    public class GambleOpenedItem
    {
        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("item_id")]
        public int ItemId { get; set; }

        [JsonProperty("quantity")]
        public int Quantity { get; set; }

        [JsonProperty("value")]
        public long Value { get; set; }
    }

    public class Recipe
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("name")]
        public string Name { get; set; }

        [JsonProperty("public_name")]
        public string PublicName { get; set; }

        [JsonProperty("inputs")]
        public List<RecipeItem> Inputs { get; set; }

        [JsonProperty("outputs")]
        public List<RecipeItem> Outputs { get; set; }
    }

    public class RecipeItem
    {
        [JsonProperty("item_id")]
        public string ItemId { get; set; }

        [JsonProperty("name")]
        public string Name { get; set; }

        [JsonProperty("quantity")]
        public int Quantity { get; set; }
    }

    // --- Account Linking Models ---

    public class LinkingStatus
    {
        [JsonProperty("is_linked")]
        public bool IsLinked { get; set; }

        [JsonProperty("linked_platforms")]
        public List<string> LinkedPlatforms { get; set; }
    }

    public class VersionInfo
    {
        [JsonProperty("version")]
        public string Version { get; set; }

        [JsonProperty("go_version")]
        public string GoVersion { get; set; }

        [JsonProperty("build_time")]
        public string BuildTime { get; set; }

        [JsonProperty("git_commit")]
        public string GitCommit { get; set; }
    }

    public class RecipeListResponse
    {
        [JsonProperty("recipes")]
        public List<Recipe> Recipes { get; set; }
    }

    // --- Expedition Models ---

    public class StartExpeditionResponse
    {
        [JsonProperty("message")]
        public string Message { get; set; }

        [JsonProperty("expedition_id")]
        public string ExpeditionId { get; set; }

        [JsonProperty("join_deadline")]
        public string JoinDeadline { get; set; }
    }

    public class ExpeditionStatus
    {
        [JsonProperty("has_active")]
        public bool HasActive { get; set; }

        [JsonProperty("active_details")]
        public ExpeditionDetails ActiveDetails { get; set; }

        [JsonProperty("cooldown_expires")]
        public string CooldownExpires { get; set; }

        [JsonProperty("on_cooldown")]
        public bool OnCooldown { get; set; }
    }

    public class ExpeditionDetails
    {
        [JsonProperty("expedition")]
        public Expedition Expedition { get; set; }

        [JsonProperty("participants")]
        public List<ExpeditionParticipant> Participants { get; set; }
    }

    public class Expedition
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("initiator_id")]
        public string InitiatorId { get; set; }

        [JsonProperty("expedition_type")]
        public string ExpeditionType { get; set; }

        [JsonProperty("state")]
        public string State { get; set; }

        [JsonProperty("created_at")]
        public DateTime CreatedAt { get; set; }

        [JsonProperty("join_deadline")]
        public DateTime JoinDeadline { get; set; }
    }

    public class ExpeditionParticipant
    {
        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("is_leader")]
        public bool IsLeader { get; set; }

        [JsonProperty("final_money")]
        public int FinalMoney { get; set; }

        [JsonProperty("final_xp")]
        public int FinalXp { get; set; }

        [JsonProperty("final_items")]
        public List<string> FinalItems { get; set; }
    }

    public class ExpeditionJournalEntry
    {
        [JsonProperty("turn_number")]
        public int TurnNumber { get; set; }

        [JsonProperty("encounter_type")]
        public string EncounterType { get; set; }

        [JsonProperty("outcome")]
        public string Outcome { get; set; }

        [JsonProperty("narrative")]
        public string Narrative { get; set; }

        [JsonProperty("fatigue")]
        public int Fatigue { get; set; }

        [JsonProperty("purse")]
        public int Purse { get; set; }
    }
}

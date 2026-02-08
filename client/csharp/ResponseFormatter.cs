using System;
using System.Collections.Generic;

namespace BrandishBot.Client
{
    /// <summary>
    /// Helper class for formatting JSON responses into user-friendly strings
    /// </summary>
    public static class ResponseFormatter
    {
        /// <summary>
        /// Map node size to human-readable unlock duration
        /// </summary>
        private static string FormatSizeToDuration(string size)
        {
            if (string.IsNullOrEmpty(size))
                return "Unknown";

            switch (size.ToLower())
            {
                case "small":
                    return "Short";
                case "medium":
                    return "Medium";
                case "large":
                    return "Long";
                default:
                    return "Mystery";
            }
        }

        /// <summary>
        /// Format price response as "Type prices: item1: price1, item2: price2, ..."
        /// Parses JSON price arrays and formats them into a readable string
        /// </summary>
        /// <param name="jsonResponse">JSON array of items with public_name and base_value</param>
        /// <param name="priceType">Type of prices (e.g., "Buy", "Sell", "Store")</param>
        /// <returns>Formatted price string</returns>
        /// <summary>
        /// Format price response using typed list
        /// </summary>
        public static string FormatPrices(List<Item> items, string priceType = "Store")
        {
            if (items == null || items.Count == 0)
                return $"{priceType} prices: (none available)";

            var prices = items.ConvertAll(i => $"{i.PublicName}: {i.BaseValue}");
            return $"{priceType} prices: " + string.Join(", ", prices);
        }

        [Obsolete("Use FormatPrices(List<Item>) instead")]
        public static string FormatPrices(string jsonResponse, string priceType = "Store")
        {
             return FormatPrices(Newtonsoft.Json.JsonConvert.DeserializeObject<List<Item>>(jsonResponse), priceType);
        }

        /// <summary>
        /// Format recipes list
        /// </summary>
        public static string FormatRecipes(List<Recipe> recipes)
        {
            if (recipes == null || recipes.Count == 0)
                return "No recipes available";

            var formatted = recipes.ConvertAll(r => r.PublicName ?? r.Name);
            return "Available recipes: " + string.Join(", ", formatted);
        }

        /// <summary>
        /// Format inventory response for readability
        /// </summary>
        public static string FormatInventory(GetInventoryResponse inventory)
        {
            if (inventory?.Items == null || inventory.Items.Count == 0)
                return "Empty inventory";

            var formattedItems = new List<string>();
            foreach (var item in inventory.Items)
            {
                if (item.Name == "money")
                    formattedItems.Insert(0, $"💰 {item.Quantity}");
                else
                    formattedItems.Add($"{item.Quantity}x {item.Name}");
            }

            return string.Join(" ", formattedItems);
        }

        // Keep legacy for backward compatibility if needed, but mark as obsolete
        [Obsolete("Use FormatInventory(GetInventoryResponse) instead")]
        public static string FormatInventory(string jsonResponse)
        {
             return FormatInventory(Newtonsoft.Json.JsonConvert.DeserializeObject<GetInventoryResponse>(jsonResponse));
        }

        /// <summary>
        /// Extract and return just the version field from version JSON response
        /// </summary>
        /// <param name="jsonResponse">JSON object with version, go_version, build_time, git_commit fields</param>
        /// <returns>Version string only</returns>
        public static string FormatVersion(VersionInfo info)
        {
            return info?.Version ?? "unknown";
        }

        [Obsolete("Use FormatVersion(VersionInfo) instead")]
        public static string FormatVersion(string jsonResponse)
        {
            try
            {
                var versionInfo = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
                return versionInfo["version"]?.ToString() ?? "unknown";
            }
            catch (Exception ex)
            {
                return $"Error parsing version: {ex.Message}";
            }
        }

        /// <summary>
        /// Default formatter that extracts and returns just the message field from any JSON response
        /// </summary>
        /// <param name="jsonResponse">JSON object containing a message field</param>
        /// <returns>The message field value, or error message if parsing fails</returns>
        /// <summary>
        /// Extract message using typed object
        /// </summary>
        public static string FormatMessage(SuccessResponse response)
        {
            return response?.Message ?? "(no message)";
        }

        /// <summary>
        /// Extract message from gamble response
        /// </summary>
        public static string FormatMessage(StartGambleResponse response)
        {
            return response?.Message ?? "(no message)";
        }

        [Obsolete("Use FormatMessage(SuccessResponse) instead")]
        public static string FormatMessage(string jsonResponse)
        {
            try
            {
                var response = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
                return response["message"]?.ToString() ?? "(no message)";
            }
            catch (Exception ex)
            {
                return $"Error parsing response: {ex.Message}";
            }
        }

        /// <summary>
        /// Format voting session options into a readable string
        /// Format: "display_name(target_level) - Unlock Time: duration Votes: vote_count |"
        /// Target level is omitted if it is 1
        /// </summary>
        /// <param name="jsonResponse">JSON object with options array</param>
        /// <returns>Formatted voting options string</returns>
        /// <summary>
        /// Format voting session options into a readable string
        /// </summary>
        public static string FormatVotingOptions(VotingSession session)
        {
            if (session?.Options == null || session.Options.Count == 0)
                return "(no options available)";

            var formattedOptions = new List<string>();
            for (int i = 0; i < session.Options.Count; i++)
            {
                var option = session.Options[i];
                if (option.NodeDetails == null) continue;

                string displayName = option.NodeDetails.DisplayName ?? "Unknown";
                string duration = FormatSizeToDuration(option.NodeDetails.Size);
                string levelStr = option.TargetLevel != 1 ? $"({option.TargetLevel})" : "";
                
                formattedOptions.Add($"{i + 1}) {displayName}{levelStr} - Unlock Time: {duration} Votes: {option.VoteCount} |");
            }

            return string.Join(" ", formattedOptions);
        }

        [Obsolete("Use FormatVotingOptions(VotingSession) instead")]
        public static string FormatVotingOptions(string jsonResponse)
        {
             return FormatVotingOptions(Newtonsoft.Json.JsonConvert.DeserializeObject<VotingSession>(jsonResponse));
        }

        /// <summary>
        /// Format unlock progress into a readable string
        /// Format: "Unlocking [node_name]: [contributions]/[target] ([percentage]%)"
        /// Or "No active unlock progress" if none
        /// </summary>
        /// <param name="jsonResponse">JSON object with unlock progress data</param>
        /// <returns>Formatted unlock progress string</returns>
        /// <summary>
        /// Format unlock progress into a readable string
        /// </summary>
        public static string FormatUnlockProgress(UnlockProgress progress)
        {
            if (progress == null || string.IsNullOrEmpty(progress.TargetNodeName))
                return "No active unlock progress";

            int barLength = 10;
            int filled = (int)(progress.CompletionPercentage / 10);
            if (filled > barLength) filled = barLength;
            string progressBar = new string('█', filled) + new string('░', barLength - filled);

            return $"Unlocking {progress.TargetNodeName}: {progress.ContributionsAccumulated}/{progress.TargetUnlockCost} ({progress.CompletionPercentage:F1}%) [{progressBar}]";
        }

        [Obsolete("Use FormatUnlockProgress(UnlockProgress) instead")]
        public static string FormatUnlockProgress(string jsonResponse)
        {
            try
            {
                var response = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
                if (response["progress"] == null && response["message"] != null) return response["message"].ToString();
                return FormatUnlockProgress(Newtonsoft.Json.JsonConvert.DeserializeObject<UnlockProgress>(jsonResponse));
            }
            catch (Exception ex)
            {
                return $"Error formatting unlock progress: {ex.Message}";
            }
        }
        /// <summary>
        /// Format progression status JSON response for readability
        /// Format: Progression: [unlocked]/[total] | Session: [node_keys] ([time] [days]d ago) | Unlock: [node_id/key] ([time] [days]d ago)
        /// </summary>
        /// <summary>
        /// Format progression status response
        /// </summary>
        public static string FormatProgressionStatus(ProgressionStatus status)
        {
            if (status == null) return "Status unavailable";

            var parts = new List<string>
            {
                $"Progression: {status.TotalUnlocked}/{status.TotalNodes}"
            };

            if (status.ActiveSession != null)
            {
                string startedAt = FormatShortTimestamp(status.ActiveSession.StartedAt);
                var nodeKeys = status.ActiveSession.Options?.ConvertAll(o => o.NodeKey) ?? new List<string>();
                string keysStr = nodeKeys.Count > 0 ? string.Join(", ", nodeKeys) : "unknown";
                parts.Add($"Session: {keysStr} ({startedAt})");
            }

            if (status.ActiveUnlockProgress != null)
            {
                string startedAt = FormatShortTimestamp(status.ActiveUnlockProgress.StartedAt);
                string identifier = status.ActiveUnlockProgress.NodeKey ?? status.ActiveUnlockProgress.NodeId ?? "unknown";
                parts.Add($"Unlock: {identifier} ({startedAt})");
            }

            return string.Join(" | ", parts);
        }

        [Obsolete("Use FormatProgressionStatus(ProgressionStatus) instead")]
        public static string FormatProgressionStatus(string jsonResponse)
        {
             return FormatProgressionStatus(Newtonsoft.Json.JsonConvert.DeserializeObject<ProgressionStatus>(jsonResponse));
        }

        /// <summary>
        /// Shorten timestamps to "HH:mm Xd ago"
        /// </summary>
        /// <summary>
        /// Shorten timestamps to "HH:mm Xd ago"
        /// </summary>
        private static string FormatShortTimestamp(DateTime dt)
        {
            DateTime utcDt = dt.ToUniversalTime();
            TimeSpan diff = DateTime.UtcNow - utcDt;
            int days = (int)Math.Floor(diff.TotalDays);
            return $"{utcDt:HH:mm} {days}d ago";
        }

        [Obsolete("Use FormatShortTimestamp(DateTime) instead")]
        private static string FormatShortTimestamp(string isoTimestamp)
        {
            if (string.IsNullOrEmpty(isoTimestamp)) return "n/a";
            if (DateTime.TryParse(isoTimestamp, null, System.Globalization.DateTimeStyles.AssumeUniversal, out DateTime dt))
            {
                return FormatShortTimestamp(dt);
            }
            return isoTimestamp;
        }

        /// <summary>
        /// Format leaderboard entries
        /// </summary>
        public static string FormatLeaderboard(List<LeaderboardEntry> entries, string metric)
        {
            if (entries == null || entries.Count == 0)
                return $"Leaderboard for {metric} is empty.";

            var lines = new List<string>();
            for (int i = 0; i < entries.Count; i++)
            {
                var entry = entries[i];
                lines.Add($"{i + 1}. {entry.Username}: {entry.Count}");
            }

            return $"Leaderboard [{metric}]: " + string.Join(" | ", lines);
        }
        /// <summary>
        /// Format account linking status
        /// </summary>
        public static string FormatLinkingStatus(LinkingStatus status)
        {
            if (status == null || !status.IsLinked || status.LinkedPlatforms == null || status.LinkedPlatforms.Count == 0)
                return "Account is not linked to any other platform.";

            return $"Account linked to: {string.Join(", ", status.LinkedPlatforms)}";
        }

        /// <summary>
        /// Format prediction result
        /// </summary>
        public static string FormatPredictionResult(PredictionResult result)
        {
            if (result == null) return "Prediction processed (no details).";

            var parts = new List<string> { result.Message };
            if (result.ContributionAwarded > 0)
            {
                parts.Add($"Awarded {result.ContributionAwarded} contributions");
            }
            if (result.WinnerXpAwarded > 0)
            {
                parts.Add($"+{result.WinnerXpAwarded} XP");
            }

            return string.Join(" | ", parts);
        }

        /// <summary>
        /// Format info response - just returns the description field
        /// </summary>
        public static string FormatInfo(InfoResponse info)
        {
            return info?.Description ?? "(no info available)";
        }
    }
}

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

            var formatted = recipes.ConvertAll(r => !string.IsNullOrEmpty(r.PublicName) ? r.PublicName : r.InternalName);
            return "Available recipes: " + string.Join(", ", formatted);
        }

        /// <summary>
        /// Format inventory response for readability
        /// </summary>
        public static string FormatInventory(GetInventoryResponse inventory)
        {
            if (inventory?.Items == null || inventory.Items.Count == 0)
                return "Empty inventory";

            var mergedItems = new Dictionary<string, int>();
            int money = 0;

            foreach (var item in inventory.Items)
            {
                if (item.InternalName == "item_money" || item.PublicName?.ToLower() == "money")
                {
                    money += item.Quantity;
                    continue;
                }

                string displayName = !string.IsNullOrEmpty(item.PublicName) ? item.PublicName : item.InternalName;
                if (string.IsNullOrEmpty(displayName)) displayName = "Unknown Item";

                if (mergedItems.ContainsKey(displayName))
                    mergedItems[displayName] += item.Quantity;
                else
                    mergedItems[displayName] = item.Quantity;
            }

            var sections = new List<string>();
            if (money > 0)
                sections.Add($"💰 {money}");

            if (mergedItems.Count > 0)
            {
                var sortedNames = new List<string>(mergedItems.Keys);
                sortedNames.Sort();
                var items = new List<string>();
                foreach (var name in sortedNames)
                {
                    items.Add($"{mergedItems[name]}x {name}");
                }
                sections.Add(string.Join(", ", items));
            }

            return sections.Count > 0 ? string.Join(" | ", sections) : "Empty inventory";
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
            try
            {
                // Try to handle wrapped response first
                var response = Newtonsoft.Json.JsonConvert.DeserializeObject<VotingSessionResponse>(jsonResponse);
                if (response?.Session != null)
                {
                    return FormatVotingOptions(response.Session);
                }
            }
            catch { }

            // Fallback to direct VotingSession deserialization for backward compatibility
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
                var nodeKeys = status.ActiveSession.Options?.ConvertAll(o => o.NodeDetails?.NodeKey ?? "unknown") ?? new List<string>();
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
        /// Format contribution leaderboard entries
        /// </summary>
        public static string FormatLeaderboard(List<ContributionLeaderboardEntry> entries)
        {
            if (entries == null || entries.Count == 0)
                return "Contribution leaderboard is empty.";

            var lines = new List<string>();
            foreach (var entry in entries)
            {
                lines.Add($"{entry.Rank}. {entry.Username}: {entry.TotalContribution}");
            }

            return "🏆 Contribution Leaderboard: " + string.Join(" | ", lines);
        }

        /// <summary>
        /// Format active quests
        /// </summary>
        public static string FormatActiveQuests(List<Quest> quests)
        {
            if (quests == null || quests.Count == 0)
                return "No active quests available.";

            var formatted = quests.ConvertAll(q => $"{q.DisplayName} ({q.QuestKey})");
            return "Active Quests: " + string.Join(" | ", formatted);
        }

        /// <summary>
        /// Format quest progress
        /// </summary>
        public static string FormatQuestProgress(List<UserQuestProgress> progressList)
        {
            if (progressList == null || progressList.Count == 0)
                return "No active quest progress found.";

            var entries = new List<string>();
            foreach (var progress in progressList)
            {
                var parts = new List<string>();
                if (progress.Progress != null)
                {
                    foreach (var kvp in progress.Progress)
                    {
                        parts.Add($"{kvp.Key}: {kvp.Value}");
                    }
                }
                string status = progress.Status ?? "unknown";
                entries.Add($"{progress.QuestKey} [{status}] (" + string.Join(", ", parts) + ")");
            }

            return "Quest Progress: " + string.Join(" | ", entries);
        }

        /// <summary>
        /// Format harvest result
        /// </summary>
        public static string FormatHarvest(HarvestResponse result)
        {
            if (result == null) return "Harvest processed.";
            
            if (result.ItemsGained == null || result.ItemsGained.Count == 0)
                return result.Message ?? "Nothing to harvest right now.";

            var items = new List<string>();
            foreach (var kvp in result.ItemsGained)
            {
                items.Add($"{kvp.Value}x {kvp.Key}");
            }

            return $"{result.Message} Gained: " + string.Join(", ", items);
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
        /// Format link initiation response
        /// </summary>
        public static string FormatLinkInitiate(LinkInitiateResponse response)
        {
            if (response == null || string.IsNullOrEmpty(response.Token))
                return "Failed to initiate linking: No token received.";

            string expireMsg = response.ExpiresIn > 0 ? $" (Expires in {response.ExpiresIn / 60}m)" : "";
            return $"Linking code: {response.Token}{expireMsg}. Run '!claimCode {response.Token}' on your other platform.";
        }

        /// <summary>
        /// Format link claim response
        /// </summary>
        public static string FormatLinkClaim(LinkClaimResponse response)
        {
            if (response == null) return "Claim request failed.";
            if (response.AwaitingConfirmation)
            {
                return $"Code claimed! Please return to {response.SourcePlatform} and run '!confirmLink' to complete the process.";
            }
            return "Code claimed successfully.";
        }

        /// <summary>
        /// Format link confirmation response
        /// </summary>
        public static string FormatLinkConfirm(LinkConfirmResponse response)
        {
            if (response == null || !response.Success)
                return "Confirmation failed. Please check the process.";

            string platforms = response.LinkedPlatforms != null ? string.Join(", ", response.LinkedPlatforms) : "none";
            return $"Account linked successfully! Current platforms: {platforms}";
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

        /// <summary>
        /// Format user jobs response for readability
        /// </summary>
        public static string FormatUserJobs(GetUserJobsResponse response)
        {
            if (response?.Jobs == null || response.Jobs.Count == 0)
                return "No job progress found.";

            var jobStrings = new List<string>();
            foreach (var job in response.Jobs)
            {
                string jobStr = $"{job.DisplayName} (Lv {job.Level})";
                if (response.PrimaryJob != null && response.PrimaryJob.JobKey == job.JobKey)
                {
                    jobStr = "⭐ " + jobStr;
                }
                jobStrings.Add(jobStr);
            }

            return "Jobs: " + string.Join(" | ", jobStrings);
        }

        /// <summary>
        /// Format health response
        /// </summary>
        public static string FormatHealth(HealthResponse response)
        {
            if (response == null) return "Unknown health status";
            string msg = response.Status;
            if (!string.IsNullOrEmpty(response.Message))
                msg += $": {response.Message}";
            return "Server Status: " + msg;
        }

        /// <summary>
        /// Format active gamble details
        /// </summary>
        public static string FormatGamble(Gamble gamble)
        {
            if (gamble == null) return "No active gamble.";

            string state = gamble.State ?? "unknown";
            int participantCount = gamble.Participants?.Count ?? 0;
            string details = $"Gamble {gamble.Id.Substring(0, 8)} [{state}] | Participants: {participantCount}";

            if (state == "open")
            {
                TimeSpan timeLeft = gamble.JoinDeadline - DateTime.UtcNow;
                if (timeLeft.TotalSeconds > 0)
                {
                    details += $" | Join time left: {(int)timeLeft.TotalSeconds}s";
                }
                else
                {
                    details += " | Joining ended";
                }
            }
            else if (state == "completed")
            {
                string winner = !string.IsNullOrEmpty(gamble.WinnerUsername) ? gamble.WinnerUsername : gamble.WinnerId;
                details += $" | Winner: {winner} | Total Value: {gamble.TotalValue}";
            }

            return details;
        }

        /// <summary>
        /// Format gamble result details
        /// </summary>
        public static string FormatGambleResult(GambleResult result)
        {
            if (result == null) return "Gamble details unavailable.";

            string winner = !string.IsNullOrEmpty(result.WinnerUsername) ? result.WinnerUsername : result.WinnerId;
            return $"🏆 Gamble Winner: {winner} | Total Value: {result.TotalValue} | Items: {result.Items?.Count ?? 0}";
        }

        /// <summary>
        /// Format gamble completed payload V2
        /// </summary>
        public static string FormatGambleCompleted(GambleCompletedPayloadV2 payload)
        {
            if (payload == null) return "Gamble completed.";

            string winner = !string.IsNullOrEmpty(payload.WinnerUsername) ? payload.WinnerUsername : payload.WinnerId;
            return $"🏆 {winner} won the gamble! Total value: {payload.TotalValue}. Participants: {payload.ParticipantCount}";
        }
    }
}

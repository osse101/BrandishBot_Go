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
        /// Format price response as "Type prices: item1: price1, item2: price2, ..."
        /// Parses JSON price arrays and formats them into a readable string
        /// </summary>
        /// <param name="jsonResponse">JSON array of items with public_name and base_value</param>
        /// <param name="priceType">Type of prices (e.g., "Buy", "Sell", "Store")</param>
        /// <returns>Formatted price string</returns>
        public static string FormatPrices(string jsonResponse, string priceType = "Store")
        {
            try
            {
                // Simple JSON parsing without Newtonsoft - parse manually
                // Expected format: [{"public_name":"item1","base_value":100},{"public_name":"item2","base_value":200}]
                
                if (string.IsNullOrEmpty(jsonResponse) || jsonResponse == "[]")
                {
                    return $"{priceType} prices: (none available)";
                }

                var prices = new List<string>();
                
                // Remove brackets and split by },{
                var cleaned = jsonResponse.Trim('[', ']', ' ');
                if (string.IsNullOrEmpty(cleaned))
                {
                    return $"{priceType} prices: (none available)";
                }

                var items = cleaned.Split(new[] { "},{" }, StringSplitOptions.None);
                
                foreach (var item in items)
                {
                    var itemCleaned = item.Trim('{', '}', ' ');
                    string publicName = null;
                    string baseValue = null;
                    
                    // Parse public_name and base_value
                    var parts = itemCleaned.Split(',');
                    foreach (var part in parts)
                    {
                        if (part.Contains("\"public_name\""))
                        {
                            var nameStart = part.IndexOf(":\"") + 2;
                            var nameEnd = part.IndexOf("\"", nameStart);
                            if (nameStart > 1 && nameEnd > nameStart)
                            {
                                publicName = part.Substring(nameStart, nameEnd - nameStart);
                            }
                        }
                        else if (part.Contains("\"base_value\""))
                        {
                            var valueStart = part.IndexOf(":") + 1;
                            baseValue = part.Substring(valueStart).Trim();
                        }
                    }
                    
                    if (!string.IsNullOrEmpty(publicName) && !string.IsNullOrEmpty(baseValue))
                    {
                        prices.Add($"{publicName}: {baseValue}");
                    }
                }
                
                if (prices.Count == 0)
                {
                    return $"{priceType} prices: (none available)";
                }
                
                return $"{priceType} prices: " + string.Join(", ", prices);
            }
            catch (Exception ex)
            {
                // Return raw response if parsing fails
                return $"Error formatting prices: {ex.Message}. Raw: {jsonResponse}";
            }
        }

        /// <summary>
        /// Format inventory JSON response for readability as "💰 money | qty1x item1 | qty2x item2"
        /// </summary>
        /// <param name="jsonResponse">JSON object with items array</param>
        /// <returns>Formatted inventory string</returns>
        public static string FormatInventory(string jsonResponse)
        {
            try
            {
                var inventory = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
                var items = inventory["items"];
                var formattedItems = new List<string>();

                // Parse all items
                foreach (var item in items)
                {
                    string name = item["name"].ToString();
                    int qty = (int)item["quantity"];

                    // Money gets special treatment - always first, emoji only
                    if (name == "money")
                    {
                        formattedItems.Insert(0, $"💰 {qty}");
                    }
                    else
                    {
                        formattedItems.Add($"{qty}x {name}");
                    }
                }

                return formattedItems.Count > 0 
                    ? string.Join(" | ", formattedItems)
                    : "Empty inventory";
            }
            catch (Exception ex)
            {
                return $"Error formatting inventory: {ex.Message}. Raw: {jsonResponse}";
            }
        }

        /// <summary>
        /// Extract and return just the version field from version JSON response
        /// </summary>
        /// <param name="jsonResponse">JSON object with version, go_version, build_time, git_commit fields</param>
        /// <returns>Version string only</returns>
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
        /// Format: "display_name(target_level) - Unlock Cost: unlock_cost Votes: vote_count |"
        /// Target level is omitted if it is 1
        /// </summary>
        /// <param name="jsonResponse">JSON object with options array</param>
        /// <returns>Formatted voting options string</returns>
        public static string FormatVotingOptions(string jsonResponse)
        {
            try
            {
                var count = 0;
                var session = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
                var options = session["options"];
                var formattedOptions = new List<string>();

                foreach (var option in options)
                {
                    count++;
                    var nodeDetails = option["node_details"];
                    if (nodeDetails == null)
                    {
                        continue;
                    }

                    string displayName = nodeDetails["display_name"]?.ToString() ?? "Unknown";
                    int targetLevel = (int?)option["target_level"] ?? 1;
                    int unlockTime = (int?)nodeDetails["unlock_cost"] ?? 0;
                    int voteCount = (int?)option["vote_count"] ?? 0;

                    // Build level suffix - omit if target_level is 1
                    string levelStr = targetLevel != 1 ? $"({targetLevel})" : "";

                    formattedOptions.Add($"{count}) {displayName}{levelStr} - Unlock Time: {unlockTime} Votes: {voteCount} |");
                }

                return formattedOptions.Count > 0
                    ? string.Join(" ", formattedOptions)
                    : "(no options available)";
            }
            catch (Exception ex)
            {
                return $"Error formatting voting options: {ex.Message}. Raw: {jsonResponse}";
            }
        }

        /// <summary>
        /// Format unlock progress into a readable string
        /// Format: "Unlocking [node_name]: [contributions]/[target] ([percentage]%)"
        /// Or "No active unlock progress" if none
        /// </summary>
        /// <param name="jsonResponse">JSON object with unlock progress data</param>
        /// <returns>Formatted unlock progress string</returns>
        public static string FormatUnlockProgress(string jsonResponse)
        {
            try
            {
                var response = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);

                // Check if there's a message field (no active progress)
                if (response["message"] != null && response["progress"] == null)
                {
                    return response["message"].ToString();
                }

                string targetNodeName = response["target_node_name"]?.ToString();
                int contributions = (int?)response["contributions_accumulated"] ?? 0;
                int target = (int?)response["target_unlock_cost"] ?? 0;
                double percentage = (double?)response["completion_percentage"] ?? 0.0;

                if (string.IsNullOrEmpty(targetNodeName))
                {
                    return "No active unlock progress";
                }

                // Format with progress bar
                int barLength = 10;
                int filled = (int)(percentage / 10);
                if (filled > barLength) filled = barLength;
                string progressBar = new string('█', filled) + new string('░', barLength - filled);

                return $"Unlocking {targetNodeName}: {contributions}/{target} ({percentage:F1}%) [{progressBar}]";
            }
            catch (Exception ex)
            {
                return $"Error formatting unlock progress: {ex.Message}. Raw: {jsonResponse}";
            }
        }
    }
}

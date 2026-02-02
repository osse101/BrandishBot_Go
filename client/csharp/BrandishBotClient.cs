using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;
using System.Collections.Generic;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace BrandishBot.Client
{
    /// <summary>
    /// Error response from API
    /// </summary>
    public class ApiErrorResponse
    {
        [JsonProperty("error")]
        public string Error { get; set; }

        [JsonProperty("fields")]
        public Dictionary<string, string> Fields { get; set; }
    }

    /// <summary>
    /// BrandishBot API Client for streamer.bot
    /// C# 4.8 compatible HTTP client for Twitch and YouTube integrations
    /// Singleton pattern: Initialize once with Initialize(), then use Instance everywhere
    /// </summary>
    public class BrandishBotClient
    {
        private static BrandishBotClient _instance;
        private static readonly object _lock = new object();

        private readonly string _baseUrl;
        private readonly string _apiKey;
        private readonly HttpClient _httpClient;

        /// <summary>
        /// Gets the singleton instance of BrandishBotClient
        /// Returns null if not initialized - check before use or call Initialize() first
        /// </summary>
        public static BrandishBotClient Instance
        {
            get { return _instance; }
        }

        /// <summary>
        /// Check if the client has been initialized
        /// </summary>
        public static bool IsInitialized
        {
            get { return _instance != null; }
        }

        /// <summary>
        /// Initialize the BrandishBot client singleton
        /// Safe to call multiple times - will reuse existing instance if config matches
        /// </summary>
        /// <param name="baseUrl">Base URL of the BrandishBot API</param>
        /// <param name="apiKey">API key for authentication</param>
        /// <param name="forceReinitialize">Force recreation even if already initialized</param>
        public static void Initialize(string baseUrl, string apiKey, bool forceReinitialize = false)
        {
            lock (_lock)
            {
                if (forceReinitialize || _instance == null)
                {
                    _instance = new BrandishBotClient(baseUrl, apiKey);
                }
            }
        }

        private readonly bool _isForwardingInstance;

        public BrandishBotClient(string baseUrl, string apiKey, bool isForwardingInstance = false)
        {
            _baseUrl = baseUrl.TrimEnd('/');
            _apiKey = apiKey;
            _isForwardingInstance = isForwardingInstance;
            _httpClient = new HttpClient();
            _httpClient.DefaultRequestHeaders.Add("X-API-Key", apiKey);
        }

        private BrandishBotClient _forwardTo;

        /// <summary>
        /// Set a secondary client to forward all requests to in the background (fire-and-forget)
        /// </summary>
        public void SetForwardingClient(BrandishBotClient devClient)
        {
            _forwardTo = devClient;
        }

        private void ForwardRequest(string method, string endpoint, Func<BrandishBotClient, Task<string>> action)
        {
            // Prevent recursion: Don't forward if we are already a forwarder
            if (_isForwardingInstance || _forwardTo == null) return;
            
            // Fire and forget in the background
            Task.Run(async () =>
            {
                try
                {
                    var response = await action(_forwardTo).ConfigureAwait(false);
                    // Log the result to streamer.bot (if we had a reference, but we use CPH.Log in the wrapper)
                    // For now, we'll just let it run. To see logs, we'd need to pass a logger or use a static delegate.
                }
                catch (Exception ex)
                {
                    // Silent fail for dev PC
                }
            });
        }

        #region Helper Methods

        private async Task<string> PostJsonAsync(string endpoint, object data)
        {
            ForwardRequest("POST", endpoint, c => c.PostJsonAsync(endpoint, data));
            var jsonBody = JsonConvert.SerializeObject(data);
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await _httpClient.PostAsync(_baseUrl + endpoint, content);
            return await HandleHttpResponse(response);
        }

        private async Task<string> PutJsonAsync(string endpoint, object data)
        {
            ForwardRequest("PUT", endpoint, c => c.PutJsonAsync(endpoint, data));
            var jsonBody = JsonConvert.SerializeObject(data);
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await _httpClient.PutAsync(_baseUrl + endpoint, content);
            return await HandleHttpResponse(response);
        }

        private async Task<string> GetAsync(string endpoint)
        {
            ForwardRequest("GET", endpoint, c => c.GetAsync(endpoint));
            var response = await _httpClient.GetAsync(_baseUrl + endpoint);
            return await HandleHttpResponse(response);
        }

        /// <summary>
        /// Get the backend version
        /// </summary>
        public async Task<string> GetVersion()
        {
            return await GetAsync("/version");
        }

        private string BuildQuery(params string[] parameters)
        {
            return "?" + string.Join("&", parameters);
        }

        /// <summary>
        /// Extract a meaningful error message from API response body
        /// </summary>
        private string ExtractErrorMessage(string responseBody, System.Net.HttpStatusCode statusCode)
        {
            if (string.IsNullOrWhiteSpace(responseBody))
            {
                return GetGenericErrorMessage(statusCode);
            }

            try
            {
                // Try to parse as JSON first
                JObject json = JObject.Parse(responseBody);

                // Check for "error" field (standard error response)
                if (json["error"] != null)
                {
                    return json["error"].Value<string>() ?? GetGenericErrorMessage(statusCode);
                }

                // Check for "message" field (alternative format)
                if (json["message"] != null)
                {
                    return json["message"].Value<string>() ?? GetGenericErrorMessage(statusCode);
                }
            }
            catch
            {
                // Not JSON, treat as plain text error message
            }

            // Use the response body as-is if it looks like an error message
            responseBody = responseBody.Trim();
            if (!string.IsNullOrEmpty(responseBody) && responseBody.Length < 500)
            {
                return responseBody;
            }

            return GetGenericErrorMessage(statusCode);
        }

        /// <summary>
        /// Get a generic error message based on HTTP status code
        /// </summary>
        private string GetGenericErrorMessage(System.Net.HttpStatusCode statusCode)
        {
            switch (statusCode)
            {
                case System.Net.HttpStatusCode.BadRequest:
                    return "Invalid request. Please check your inputs.";
                case System.Net.HttpStatusCode.Unauthorized:
                    return "Authentication failed. Please check your API key.";
                case System.Net.HttpStatusCode.Forbidden:
                    return "That feature is locked. Unlock it in the progression tree.";
                case System.Net.HttpStatusCode.NotFound:
                    return "Resource not found.";
                case System.Net.HttpStatusCode.InternalServerError:
                    return "Server error occurred. Please try again.";
                case System.Net.HttpStatusCode.ServiceUnavailable:
                    return "Server is temporarily unavailable. Please try again later.";
                default:
                    return "An error occurred. Please try again.";
            }
        }

        /// <summary>
        /// Handle HTTP response and throw with meaningful error messages on failure
        /// </summary>
        private async Task<string> HandleHttpResponse(HttpResponseMessage response)
        {
            if (response.IsSuccessStatusCode)
            {
                return await response.Content.ReadAsStringAsync();
            }

            // Extract error message from response body
            string errorBody = await response.Content.ReadAsStringAsync();
            string errorMessage = ExtractErrorMessage(errorBody, response.StatusCode);

            // Include status code in the message so it can be identified by the wrapper
            throw new HttpRequestException($"{(int)response.StatusCode} {response.StatusCode}: {errorMessage}");
        }

        #endregion

        #region General
        
        /// <summary>
        /// Get feature information
        /// </summary>
        public async Task<string> GetInfo()
        {
            return await GetAsync("/api/v1/info");
        }

        #endregion

        #region User Management

        /// <summary>
        /// Register a new user (auto-called on first interaction)
        /// </summary>
        public async Task<string> RegisterUser(string platform, string platformId, string username)
        {
            return await PostJsonAsync("/api/v1/user/register", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        #endregion

        #region Inventory Operations


        /// <summary>
        /// Get user's inventory
        /// </summary>
        public async Task<string> GetInventory(string platform, string platformId, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId,
                "username=" + username
            );
            return await GetAsync("/api/v1/user/inventory" + query);
        }

        /// <summary>
        /// Get user's inventory by username (no platformId required)
        /// </summary>
        public async Task<string> GetInventoryByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync("/api/v1/user/inventory-by-username" + query);
        }

        /// <summary>
        /// Add item by username (no platformId required)
        /// </summary>
        public async Task<string> AddItemByUsername(string platform, string username, string itemName, int quantity=1)
        {
            return await PostJsonAsync("/api/v1/user/item/add", new
            {
                platform = platform,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Remove item by username (no platformId required)
        /// </summary>
        public async Task<string> RemoveItemByUsername(string platform, string username, string itemName, int quantity=1)
        {
            return await PostJsonAsync("/api/v1/user/item/remove", new
            {
                platform = platform,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Give item from one user to another
        /// </summary>
        public async Task<string> GiveItem(string fromPlatform, string fromPlatformId, string fromUsername, 
            string toPlatform, string toUsername, string itemName, int quantity=1)
        {
            return await PostJsonAsync("/api/v1/user/item/give", new
            {
                owner_platform = fromPlatform,
                owner_platform_id = fromPlatformId,
                owner = fromUsername,
                receiver_platform = toPlatform,
                receiver = toUsername,
                item_name = itemName,
                quantity = quantity
            });
        }

        #endregion

        #region Economy

        /// <summary>
        /// Buy an item from the shop
        /// </summary>
        public async Task<string> BuyItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/buy", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Sell an item from inventory
        /// </summary>
        public async Task<string> SellItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/sell", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Get current buy prices for items
        /// </summary>
        public async Task<string> GetBuyPrices()
        {
            return await GetAsync("/api/v1/prices/buy");
        }

        /// <summary>
        /// Get current item prices (Sell Prices)
        /// </summary>
        public async Task<string> GetSellPrices()
        {
            return await GetAsync("/api/v1/prices");
        }

        #endregion

        #region Item Actions

        /// <summary>
        /// Use an item (opens lootboxes, activates items, etc.)
        /// </summary>
        public async Task<string> UseItem(string platform, string platformId, string username, 
            string itemName, int quantity=1, string targetUsername = null)
        {
            var data = new Dictionary<string, object>
            {
                { "platform", platform },
                { "platform_id", platformId },
                { "username", username },
                { "item_name", itemName },
                { "quantity", quantity }
            };

            if (!string.IsNullOrEmpty(targetUsername))
            {
                data["target_user"] = targetUsername;
            }

            return await PostJsonAsync("/api/v1/user/item/use", data);
        }

        /// <summary>
        /// Search for items (opens random lootboxes based on engagement)
        /// </summary>
        public async Task<string> Search(string platform, string platformId, string username)
        {
            return await PostJsonAsync("/api/v1/user/search", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        #endregion

        #region Crafting

        /// <summary>
        /// Upgrade an item using a recipe
        /// </summary>
        public async Task<string> UpgradeItem(string platform, string platformId, string username, string itemName, int quantity=1)
        {
            return await PostJsonAsync("/api/v1/user/item/upgrade", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Disassemble an item to get materials
        /// </summary>
        public async Task<string> DisassembleItem(string platform, string platformId, string username, string itemName, int quantity=1)
        {
            return await PostJsonAsync("/api/v1/user/item/disassemble", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Get available crafting recipes
        /// </summary>
        public async Task<string> GetRecipes()
        {
            return await GetAsync("/api/v1/recipes");
        }

        /// <summary>
        /// Get unlocked crafting recipes for a user (self-mode)
        /// </summary>
        public async Task<string> GetUnlockedRecipes(string platform, string platformId, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId,
                "user=" + username
            );
            return await GetAsync("/api/v1/recipes" + query);
        }

        /// <summary>
        /// Get unlocked crafting recipes for a user by username (target-mode)
        /// </summary>
        public async Task<string> GetUnlockedRecipesByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "user=" + username
            );
            return await GetAsync("/api/v1/recipes" + query);
        }

        #endregion

        #region Gamble System

        /// <summary>
        /// Start a new gamble session
        /// </summary>
        public async Task<string> StartGamble(string platform, string platformId, string username,
            string itemName, int quantity=1)
        {
            var responseJson = await PostJsonAsync("/api/v1/gamble/start", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                bets = new[] { new { item_name = itemName, quantity = quantity } }
            });

            // Parse the response to extract gamble_id
            try
            {
                var response = JObject.Parse(responseJson);
                return response["gamble_id"]?.Value<string>() ?? responseJson;
            }
            catch
            {
                // Fallback to raw response if parsing fails
                return responseJson;
            }
        }

        /// <summary>
        /// Join an existing gamble session
        /// </summary>
        public async Task<string> JoinGamble(string gambleId, string platform, string platformId, string username)
        {
            return await PostJsonAsync("/api/v1/gamble/join?id=" + gambleId, new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Get active gamble details
        /// </summary>
        public async Task<string> GetActiveGamble()
        {
            return await GetAsync("/api/v1/gamble/get");
        }

        #endregion

        #region Stats & Leaderboards

        /// <summary>
        /// Get user statistics (self-mode)
        /// </summary>
        public async Task<string> GetUserStats(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync("/api/v1/stats/user" + query);
        }

        /// <summary>
        /// Get user statistics by username (target-mode)
        /// </summary>
        public async Task<string> GetUserStatsByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync("/api/v1/stats/user" + query);
        }

        /// <summary>
        /// Get system-wide statistics
        /// </summary>
        public async Task<string> GetSystemStats()
        {
            return await GetAsync("/api/v1/stats/system");
        }

        /// <summary>
        /// Get leaderboard
        /// </summary>
        public async Task<string> GetLeaderboard(string metric = "engagement_score", int limit = 10)
        {
            var query = BuildQuery(
                "metric=" + metric,
                "limit=" + limit.ToString()
            );
            return await GetAsync("/api/v1/stats/leaderboard" + query);
        }

        /// <summary>
        /// Get user timeout status (check if user is timed out)
        /// Returns JSON with is_timed_out (bool) and remaining_seconds (double)
        /// </summary>
        public async Task<string> GetUserTimeout(string username)
        {
            var query = BuildQuery("username=" + username);
            return await GetAsync("/api/v1/user/timeout" + query);
        }

        #endregion

        #region Progression System

        /// <summary>
        /// Get progression tree structure
        /// </summary>
        public async Task<string> GetProgressionTree()
        {
            return await GetAsync("/api/v1/progression/tree");
        }

        /// <summary>
        /// Get available (unlockable) progression nodes
        /// </summary>
        public async Task<string> GetAvailableNodes()
        {
            return await GetAsync("/api/v1/progression/available");
        }

        /// <summary>
        /// Vote to unlock a progression node
        /// </summary>
        public async Task<string> VoteForNode(string platform, string platformId, string nodeKey)
        {
            return await PostJsonAsync("/api/v1/progression/vote", new
            {
                platform = platform,
                platform_id = platformId,
                node_key = nodeKey
            });
        }

        /// <summary>
        /// Get progression status (unlocked nodes, votes, etc.)
        /// </summary>
        public async Task<string> GetProgressionStatus()
        {
            return await GetAsync("/api/v1/progression/status");
        }

        /// <summary>
        /// Get user engagement breakdown (contribution points)
        /// </summary>
        public async Task<string> GetUserEngagement(string platform, string platformId)
        {
            var query = BuildQuery("platform=" + platform, "platform_id=" + platformId);
            return await GetAsync("/api/v1/progression/engagement" + query);
        }

        /// <summary>
        /// Get user engagement breakdown by username (contribution points)
        /// </summary>
        public async Task<string> GetUserEngagementByUsername(string platform, string username)
        {
            var query = BuildQuery("platform=" + platform, "username=" + username);
            return await GetAsync("/api/v1/progression/engagement-by-username" + query);
        }

        /// <summary>
        /// Get current voting session details
        /// </summary>
        public async Task<string> GetVotingSession()
        {
            return await GetAsync("/api/v1/progression/session");
        }

        /// <summary>
        /// Get unlock progress for the current voting session
        /// </summary>
        public async Task<string> GetUnlockProgress()
        {
            return await GetAsync("/api/v1/progression/unlock-progress");
        }

        #endregion

        #region Progression Admin

        /// <summary>
        /// Admin: Unlock a specific node at a specific level
        /// </summary>
        /// <param name="level">Target level to unlock (default: 1)</param>
        public async Task<string> AdminUnlockNode(string nodeKey, int level = 1)
        {
            return await PostJsonAsync("/api/v1/progression/admin/unlock", new { node_key = nodeKey, level = level });
        }

        /// <summary>
        /// Admin: Unlock ALL progression nodes at max level (DEBUG ONLY)
        /// </summary>
        public async Task<string> AdminUnlockAllNodes()
        {
            return await PostJsonAsync("/api/v1/progression/admin/unlock-all", new { });
        }

        /// <summary>
        /// Admin: Re-lock a specific node at a specific level
        /// </summary>
        public async Task<string> AdminRelockNode(string nodeKey, int level)
        {
            return await PostJsonAsync("/api/v1/progression/admin/relock", new { node_key = nodeKey, level = level });
        }

        /// <summary>
        /// Admin: Start a new voting session
        /// </summary>
        public async Task<string> AdminStartVoting()
        {
            return await PostJsonAsync("/api/v1/progression/admin/start-voting", new { });
        }

        /// <summary>
        /// Admin: End the current voting session
        /// </summary>
        public async Task<string> AdminEndVoting()
        {
            return await PostJsonAsync("/api/v1/progression/admin/end-voting", new { });
        }

        /// <summary>
        /// Admin: Reset the entire progression system
        /// </summary>
        public async Task<string> AdminResetProgression(string resetBy, string reason, bool preserveUserProgression)
        {
            return await PostJsonAsync("/api/v1/progression/admin/reset", new 
            { 
                reset_by = resetBy,
                reason = reason,
                preserve_user_progression = preserveUserProgression
            });
        }

        /// <summary>
        /// Admin: Add contribution points to the progression system
        /// </summary>
        public async Task<string> AdminAddContribution(int amount)
        {
            return await PostJsonAsync("/api/v1/progression/admin/contribution", new { amount = amount });
        }

        /// <summary>
        /// Admin: Reload engagement weight cache
        /// </summary>
        public async Task<string> AdminReloadWeights()
        {
            return await PostJsonAsync("/api/v1/admin/progression/reload-weights", new { });
        }

        /// <summary>
        /// Admin: Get user cache statistics
        /// </summary>
        public async Task<string> AdminGetCacheStats()
        {
            return await GetAsync("/api/v1/admin/cache/stats");
        }

        #endregion

        #region Jobs System

        /// <summary>
        /// Get user's job progress (self-mode)
        /// </summary>
        public async Task<string> GetUserJobs(string platform, string platformId, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId,
                "username=" + username
            );
            return await GetAsync("/api/v1/jobs/user" + query);
        }

        /// <summary>
        /// Get user's job progress by username (target-mode)
        /// </summary>
        public async Task<string> GetUserJobsByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync("/api/v1/jobs/user" + query);
        }

        /// <summary>
        /// Award XP to a user for a specific job (Streamer/Admin only)
        /// </summary>
        public async Task<string> AwardJobXP(string platform, string username, string jobKey, int xpAmount)
        {
            return await PostJsonAsync("/api/v1/jobs/award-xp", new
            {
                platform = platform,
                username = username,
                job_key = jobKey,
                xp_amount = xpAmount
            });
        }

        #endregion

        #region Account Linking

        /// <summary>
        /// Initiate account linking process
        /// </summary>
        public async Task<string> InitiateLinking(string platform, string platformId, string username)
        {
            return await PostJsonAsync("/api/v1/link/initiate", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Claim a linking code from another platform
        /// </summary>
        public async Task<string> ClaimLinkingCode(string platform, string platformId, string username, string code)
        {
            return await PostJsonAsync("/api/v1/link/claim", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                code = code
            });
        }

        /// <summary>
        /// Confirm account linking
        /// </summary>
        public async Task<string> ConfirmLinking(string platform, string platformId)
        {
            return await PostJsonAsync("/api/v1/link/confirm", new
            {
                platform = platform,
                platform_id = platformId
            });
        }

        /// <summary>
        /// Unlink accounts
        /// </summary>
        public async Task<string> UnlinkAccounts(string platform, string platformId, string targetPlatform)
        {
            return await PostJsonAsync("/api/v1/link/unlink", new
            {
                platform = platform,
                platform_id = platformId,
                target_platform = targetPlatform
            });
        }

        /// <summary>
        /// Get linking status for a user
        /// </summary>
        public async Task<string> GetLinkingStatus(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync("/api/v1/link/status" + query);
        }

        #endregion

        #region Message Handler (All-in-One)

        /// <summary>
        /// Handle a chat message (processes commands, tracks engagement, gives rewards)
        /// Use this for Twitch/YouTube chat integration
        /// </summary>
        public async Task<string> HandleMessage(string platform, string platformId, string username, string message)
        {
            return await PostJsonAsync("/api/v1/message/handle", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                message = message
            });
        }

        #endregion

        #region Timeout Management

        /// <summary>
        /// Get user's timeout status
        /// </summary>
        /// <param name="platform">Platform (twitch, youtube, discord)</param>
        /// <param name="username">Username to check</param>
        /// <returns>JSON with platform, username, is_timed_out, remaining_seconds</returns>
        public async Task<string> GetUserTimeout(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync("/api/v1/user/timeout" + query);
        }

        /// <summary>
        /// Set or extend a user's timeout (accumulates with existing timeout)
        /// </summary>
        /// <param name="platform">Platform (twitch, youtube, discord)</param>
        /// <param name="username">Username to timeout</param>
        /// <param name="durationSeconds">Duration in seconds (1-86400)</param>
        /// <param name="reason">Optional reason for the timeout</param>
        public async Task<string> SetUserTimeoutAsync(string platform, string username, int durationSeconds, string reason = null)
        {
            var body = new
            {
                platform = platform,
                username = username,
                duration_seconds = durationSeconds,
                reason = reason ?? ""
            };
            return await PutJsonAsync("/api/v1/user/timeout", body);
        }

        /// <summary>
        /// Admin: Clear a user's timeout
        /// </summary>
        /// <param name="platform">Platform (twitch, youtube, discord)</param>
        /// <param name="username">Username to clear timeout for</param>
        public async Task<string> AdminClearTimeoutAsync(string platform, string username)
        {
            return await PostJsonAsync("/api/v1/admin/timeout/clear", new
            {
                platform = platform,
                username = username
            });
        }

        #endregion

        #region Health Checks

        /// <summary>
        /// Check if API is alive
        /// </summary>
        public async Task<string> HealthCheck()
        {
            return await GetAsync("/healthz");
        }

        /// <summary>
        /// Check if API is ready (includes DB check)
        /// </summary>
        public async Task<string> ReadyCheck()
        {
            return await GetAsync("/readyz");
        }

        #endregion
    }

    #region Platform Constants

    /// <summary>
    /// Platform identifiers for Twitch and YouTube
    /// </summary>
    public static class Platform
    {
        public const string Twitch = "twitch";
        public const string YouTube = "youtube";
        public const string Discord = "discord";
    }

    /// <summary>
    /// Event types for stats tracking
    /// </summary>
    public static class EventType
    {
        public const string Message = "message";
        public const string Follow = "follow";
        public const string Subscribe = "subscribe";
        public const string Raid = "raid";
        public const string Bits = "bits";
        public const string Gift = "gift";
    }

    /// <summary>
    /// Item public names (command names used in chat)
    /// These are the user-facing names for items
    /// </summary>
    public static class ItemName
    {
        public const string Money = "money";
        public const string Junkbox = "junkbox";   // Tier 0 - Rusty Lootbox
        public const string Lootbox = "lootbox";   // Tier 1 - Basic Lootbox
        public const string Goldbox = "goldbox";   // Tier 2 - Golden Lootbox
        public const string Missile = "missile";   // Ray Gun / Blaster
    }

    #endregion
}

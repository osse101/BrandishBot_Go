using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;
using System.Collections.Generic;
using Newtonsoft.Json;

namespace BrandishBot.Client
{
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

        private BrandishBotClient(string baseUrl, string apiKey)
        {
            _baseUrl = baseUrl.TrimEnd('/');
            _apiKey = apiKey;
            _httpClient = new HttpClient();
            _httpClient.DefaultRequestHeaders.Add("X-API-Key", apiKey);
        }

        #region Helper Methods

        private async Task<string> PostJsonAsync(string endpoint, object data)
        {
            var jsonBody = JsonConvert.SerializeObject(data);
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await _httpClient.PostAsync(_baseUrl + endpoint, content);
            response.EnsureSuccessStatusCode();
            return await response.Content.ReadAsStringAsync();
        }

        private async Task<string> GetAsync(string endpoint)
        {
            var response = await _httpClient.GetAsync(_baseUrl + endpoint);
            response.EnsureSuccessStatusCode();
            return await response.Content.ReadAsStringAsync();
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
        /// Add item to user's inventory (Admin/Streamer only)
        /// </summary>
        /// <summary>
        /// Add item to user's inventory (Admin/Streamer only)
        /// </summary>
        public async Task<string> AddItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/add", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Add item by username (no platformId required)
        /// </summary>
        public async Task<string> AddItemByUsername(string platform, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/add-by-username", new
            {
                platform = platform,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Remove item from user's inventory (Admin/Streamer only)
        /// </summary>
        /// <summary>
        /// Remove item from user's inventory (Admin/Streamer only)
        /// </summary>
        public async Task<string> RemoveItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/remove", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Remove item by username (no platformId required)
        /// </summary>
        public async Task<string> RemoveItemByUsername(string platform, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/remove-by-username", new
            {
                platform = platform,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Use an item from inventory (with optional target user)
        /// </summary>
        /// <summary>
        /// Remove item from user's inventory (Admin/Streamer only)
        /// </summary>
        public async Task<string> RemoveItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/remove", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Give item from one user to another
        /// </summary>
        public async Task<string> GiveItem(string fromPlatform, string fromPlatformId, 
            string toPlatform, string toPlatformId, string toUsername, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/give", new
            {
                from_platform = fromPlatform,
                from_platform_id = fromPlatformId,
                to_platform = toPlatform,
                to_platform_id = toPlatformId,
                to_username = toUsername,
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
            string itemName, int quantity, string targetUsername = null)
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
        public async Task<string> UpgradeItem(string platform, string platformId, string username, string itemName, int quantity)
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
        public async Task<string> DisassembleItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/user/item/disassemble", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
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
        /// Get unlocked crafting recipes for a user
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

        #endregion

        #region Gamble System

        /// <summary>
        /// Start a new gamble session
        /// </summary>
        public async Task<string> StartGamble(string platform, string platformId, string username, 
            string itemName, int quantity)
        {
            return await PostJsonAsync("/api/v1/gamble/start", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                bets = new[] { new { item_name = itemName, quantity = quantity } }
            });
        }

        /// <summary>
        /// Join an existing gamble session
        /// </summary>
        public async Task<string> JoinGamble(string platform, string platformId, string username, 
            string gambleId, string itemName, int quantity)
        {
            var content = new StringContent(
                JsonConvert.SerializeObject(new
                {
                    platform = platform,
                    platform_id = platformId,
                    username = username,
                    bets = new[] { new { item_name = itemName, quantity = quantity } }
                }),
                Encoding.UTF8,
                "application/json"
            );
            
            var response = await _httpClient.PostAsync(
                _baseUrl + "/api/v1/gamble/join?id=" + gambleId,
                content
            );
            response.EnsureSuccessStatusCode();
            return await response.Content.ReadAsStringAsync();
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
        /// Record a user event (message, follow, sub, etc.)
        /// </summary>
        public async Task<string> RecordEvent(string platform, string platformId, string eventType, object metadata = null)
        {
            return await PostJsonAsync("/api/v1/stats/event", new
            {
                platform = platform,
                platform_id = platformId,
                event_type = eventType,
                metadata = metadata
            });
        }

        /// <summary>
        /// Get user statistics
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
        public async Task<string> GetUserEngagement(string userId)
        {
            var query = BuildQuery("user_id=" + userId);
            return await GetAsync("/api/v1/progression/engagement" + query);
        }

        /// <summary>
        /// Get contribution leaderboard
        /// </summary>
        public async Task<string> GetContributionLeaderboard()
        {
            return await GetAsync("/api/v1/progression/leaderboard");
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
        /// Admin: Re-lock a specific node at a specific level
        /// </summary>
        public async Task<string> AdminRelockNode(string nodeKey, int level)
        {
            return await PostJsonAsync("/api/v1/progression/admin/relock", new { node_key = nodeKey, level = level });
        }

        /// <summary>
        /// Admin: Instantly unlock the current vote leader without waiting
        /// </summary>
        public async Task<string> AdminInstantUnlock()
        {
            return await PostJsonAsync("/api/v1/progression/admin/instant-unlock", new { });
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

        #endregion

        #region Jobs System

        /// <summary>
        /// Get all available jobs
        /// </summary>
        public async Task<string> GetAllJobs()
        {
            return await GetAsync("/api/v1/jobs");
        }

        /// <summary>
        /// Get user's job progress
        /// </summary>
        public async Task<string> GetUserJobs(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync("/api/v1/jobs/user" + query);
        }

        /// <summary>
        /// Award XP to a user for a specific job (Streamer/Admin only)
        /// </summary>
        public async Task<string> AwardJobXP(string platform, string platformId, string username, string jobName, int xpAmount)
        {
            return await PostJsonAsync("/api/v1/jobs/award-xp", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                job_name = jobName,
                xp_amount = xpAmount
            });
        }

        /// <summary>
        /// Get the active bonus for a job (e.g., search_bonus for Scavenger job)
        /// </summary>
        public async Task<string> GetJobBonus(string userId, string jobKey, string bonusType)
        {
            var query = BuildQuery(
                "user_id=" + userId,
                "job_key=" + jobKey,
                "bonus_type=" + bonusType
            );
            return await GetAsync("/api/v1/jobs/bonus" + query);
        }

        /// <summary>
        /// Admin: Award XP to a user for a specific job
        /// </summary>
        public async Task<string> AdminAwardXP(string platform, string username, string jobKey, int amount)
        {
            return await PostJsonAsync("/api/v1/admin/job/award-xp", new
            {
                platform = platform,
                username = username,
                job_key = jobKey,
                amount = amount
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
        public async Task<string> UnlinkAccounts(string platform, string platformId)
        {
            return await PostJsonAsync("/api/v1/link/unlink", new
            {
                platform = platform,
                platform_id = platformId
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

        #region Economy (Extended)

        /// <summary>
        /// Get current buy prices for items
        /// </summary>
        public async Task<string> GetBuyPrices()
        {
            return await GetAsync("/api/v1/prices/buy");
        }

        #endregion

        #region Admin Utilities

        /// <summary>
        /// Reload item name aliases from configuration (Admin only)
        /// </summary>
        public async Task<string> ReloadAliases()
        {
            return await PostJsonAsync("/api/v1/admin/reload-aliases", new { });
        }

        /// <summary>
        /// Test endpoint for debugging
        /// </summary>
        public async Task<string> Test(string platform, string platformId, string username)
        {
            return await PostJsonAsync("/api/v1/test", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
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
    /// Common item IDs (reference your database)
    /// DEPRECATED: Use ItemName constants instead. Item operations now use string names.
    /// </summary>
    [Obsolete("Use ItemName constants instead. Item operations now use string item_name parameters.")]
    public static class ItemId
    {
        public const int Money = 1;
        public const int Lootbox0 = 2;
        public const int Lootbox1 = 3;
        public const int Lootbox2 = 4;
        public const int Blaster = 5;
        // Add more as needed
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

using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

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
        /// Must call Initialize() first before accessing
        /// </summary>
        public static BrandishBotClient Instance
        {
            get
            {
                if (_instance == null)
                {
                    throw new InvalidOperationException("BrandishBotClient not initialized. Call Initialize() first.");
                }
                return _instance;
            }
        }

        /// <summary>
        /// Initialize the BrandishBot client singleton
        /// Call this once at application startup
        /// </summary>
        /// <param name="baseUrl">Base URL of the BrandishBot API</param>
        /// <param name="apiKey">API key for authentication</param>
        public static void Initialize(string baseUrl, string apiKey)
        {
            if (_instance != null)
            {
                throw new InvalidOperationException("BrandishBotClient already initialized.");
            }

            lock (_lock)
            {
                if (_instance == null)
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

        private async Task<string> PostJsonAsync(string endpoint, string jsonBody)
        {
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
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}""}}",
                platform, platformId, username);
            return await PostJsonAsync("/user/register", json);
        }

        #endregion

        #region Inventory Operations

        /// <summary>
        /// Get user's inventory
        /// </summary>
        public async Task<string> GetInventory(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync("/user/inventory" + query);
        }

        /// <summary>
        /// Add item to user's inventory (Admin/Streamer only)
        /// </summary>
        public async Task<string> AddItem(string platform, string platformId, int itemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""item_id"":{2},""quantity"":{3}}}",
                platform, platformId, itemId, quantity);
            return await PostJsonAsync("/user/item/add", json);
        }

        /// <summary>
        /// Remove item from user's inventory (Admin/Streamer only)
        /// </summary>
        public async Task<string> RemoveItem(string platform, string platformId, int itemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""item_id"":{2},""quantity"":{3}}}",
                platform, platformId, itemId, quantity);
            return await PostJsonAsync("/user/item/remove", json);
        }

        /// <summary>
        /// Give item from one user to another
        /// </summary>
        public async Task<string> GiveItem(string fromPlatform, string fromPlatformId, 
            string toPlatform, string toPlatformId, string toUsername, int itemId, int quantity)
        {
            var json = string.Format(@"{{""from_platform"":""{0}"",""from_platform_id"":""{1}"",""to_platform"":""{2}"",""to_platform_id"":""{3}"",""to_username"":""{4}"",""item_id"":{5},""quantity"":{6}}}",
                fromPlatform, fromPlatformId, toPlatform, toPlatformId, toUsername, itemId, quantity);
            return await PostJsonAsync("/user/item/give", json);
        }

        #endregion

        #region Economy

        /// <summary>
        /// Buy an item from the shop
        /// </summary>
        public async Task<string> BuyItem(string platform, string platformId, string username, int itemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""item_id"":{3},""quantity"":{4}}}",
                platform, platformId, username, itemId, quantity);
            return await PostJsonAsync("/user/item/buy", json);
        }

        /// <summary>
        /// Sell an item from inventory
        /// </summary>
        public async Task<string> SellItem(string platform, string platformId, string username, int itemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""item_id"":{3},""quantity"":{4}}}",
                platform, platformId, username, itemId, quantity);
            return await PostJsonAsync("/user/item/sell", json);
        }

        /// <summary>
        /// Get current item prices
        /// </summary>
        public async Task<string> GetPrices()
        {
            return await GetAsync("/prices");
        }

        #endregion

        #region Item Actions

        /// <summary>
        /// Use an item (opens lootboxes, activates items, etc.)
        /// </summary>
        public async Task<string> UseItem(string platform, string platformId, string username, 
            int itemId, int quantity, string targetUsername = null)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""item_id"":{3},""quantity"":{4}{5}}}",
                platform, platformId, username, itemId, quantity,
                string.IsNullOrEmpty(targetUsername) ? "" : string.Format(@",""target_username"":""{0}""", targetUsername));
            return await PostJsonAsync("/user/item/use", json);
        }

        /// <summary>
        /// Search for items (opens random lootboxes based on engagement)
        /// </summary>
        public async Task<string> Search(string platform, string platformId, string username)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}""}}",
                platform, platformId, username);
            return await PostJsonAsync("/user/search", json);
        }

        #endregion

        #region Crafting

        /// <summary>
        /// Upgrade an item using a recipe
        /// </summary>
        public async Task<string> UpgradeItem(string platform, string platformId, string username, int recipeId)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""recipe_id"":{3}}}",
                platform, platformId, username, recipeId);
            return await PostJsonAsync("/user/item/upgrade", json);
        }

        /// <summary>
        /// Disassemble an item to get materials
        /// </summary>
        public async Task<string> DisassembleItem(string platform, string platformId, string username, int itemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""item_id"":{3},""quantity"":{4}}}",
                platform, platformId, username, itemId, quantity);
            return await PostJsonAsync("/user/item/disassemble", json);
        }

        /// <summary>
        /// Get available crafting recipes
        /// </summary>
        public async Task<string> GetRecipes()
        {
            return await GetAsync("/recipes");
        }

        #endregion

        #region Gamble System

        /// <summary>
        /// Start a new gamble session
        /// </summary>
        public async Task<string> StartGamble(string platform, string platformId, string username, 
            int lootboxItemId, int quantity)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""bets"":[{{""item_id"":{3},""quantity"":{4}}}]}}",
                platform, platformId, username, lootboxItemId, quantity);
            return await PostJsonAsync("/gamble/start", json);
        }

        /// <summary>
        /// Join an existing gamble session
        /// </summary>
        public async Task<string> JoinGamble(string gambleId, string platform, string platformId, 
            string username, int lootboxItemId, int quantity)
        {
            var json = string.Format(@"{{""gamble_id"":""{0}"",""platform"":""{1}"",""platform_id"":""{2}"",""username"":""{3}"",""bets"":[{{""item_id"":{4},""quantity"":{5}}}]}}",
                gambleId, platform, platformId, username, lootboxItemId, quantity);
            return await PostJsonAsync("/gamble/join", json);
        }

        /// <summary>
        /// Get active gamble details
        /// </summary>
        public async Task<string> GetActiveGamble()
        {
            return await GetAsync("/gamble/get");
        }

        #endregion

        #region Stats & Leaderboards

        /// <summary>
        /// Record a user event (message, follow, sub, etc.)
        /// </summary>
        public async Task<string> RecordEvent(string platform, string platformId, string eventType, string metadata = null)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""event_type"":""{2}""{3}}}",
                platform, platformId, eventType,
                string.IsNullOrEmpty(metadata) ? "" : string.Format(@",""metadata"":{0}", metadata));
            return await PostJsonAsync("/stats/event", json);
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
            return await GetAsync("/stats/user" + query);
        }

        /// <summary>
        /// Get system-wide statistics
        /// </summary>
        public async Task<string> GetSystemStats()
        {
            return await GetAsync("/stats/system");
        }

        /// <summary>
        /// Get leaderboard
        /// </summary>
        public async Task<string> GetLeaderboard(string metric = "engagement_score", int limit = 10)
        {
            var query = BuildQuery(
                "metric=" + metric,
                "limit=" + limit
            );
            return await GetAsync("/stats/leaderboard" + query);
        }

        #endregion

        #region Progression System

        /// <summary>
        /// Get progression tree structure
        /// </summary>
        public async Task<string> GetProgressionTree()
        {
            return await GetAsync("/progression/tree");
        }

        /// <summary>
        /// Get available (unlockable) progression nodes
        /// </summary>
        public async Task<string> GetAvailableNodes()
        {
            return await GetAsync("/progression/available");
        }

        /// <summary>
        /// Vote to unlock a progression node
        /// </summary>
        public async Task<string> VoteForNode(string platform, string platformId, string nodeKey)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""node_key"":""{2}""}}",
                platform, platformId, nodeKey);
            return await PostJsonAsync("/progression/vote", json);
        }

        /// <summary>
        /// Get progression status (unlocked nodes, votes, etc.)
        /// </summary>
        public async Task<string> GetProgressionStatus()
        {
            return await GetAsync("/progression/status");
        }

        /// <summary>
        /// Get community engagement score
        /// </summary>
        public async Task<string> GetEngagementScore()
        {
            return await GetAsync("/progression/engagement");
        }

        /// <summary>
        /// Get contribution leaderboard
        /// </summary>
        public async Task<string> GetContributionLeaderboard()
        {
            return await GetAsync("/progression/leaderboard");
        }

        /// <summary>
        /// Get current voting session details
        /// </summary>
        public async Task<string> GetVotingSession()
        {
            return await GetAsync("/progression/session");
        }

        /// <summary>
        /// Get unlock progress for the current voting session
        /// </summary>
        public async Task<string> GetUnlockProgress()
        {
            return await GetAsync("/progression/unlock-progress");
        }

        #endregion

        #region Progression Admin

        /// <summary>
        /// Admin: Unlock a specific node
        /// </summary>
        public async Task<string> AdminUnlockNode(string nodeKey)
        {
            var json = string.Format(@"{{""node_key"":""{0}""}}", nodeKey);
            return await PostJsonAsync("/progression/admin/unlock", json);
        }

        /// <summary>
        /// Admin: Re-lock a specific node
        /// </summary>
        public async Task<string> AdminRelockNode(string nodeKey)
        {
            var json = string.Format(@"{{""node_key"":""{0}""}}", nodeKey);
            return await PostJsonAsync("/progression/admin/relock", json);
        }

        /// <summary>
        /// Admin: Instantly unlock a node without voting
        /// </summary>
        public async Task<string> AdminInstantUnlock(string nodeKey)
        {
            var json = string.Format(@"{{""node_key"":""{0}""}}", nodeKey);
            return await PostJsonAsync("/progression/admin/instant-unlock", json);
        }

        /// <summary>
        /// Admin: Start a voting session for a specific node
        /// </summary>
        public async Task<string> AdminStartVoting(string nodeKey)
        {
            var json = string.Format(@"{{""node_key"":""{0}""}}", nodeKey);
            return await PostJsonAsync("/progression/admin/start-voting", json);
        }

        /// <summary>
        /// Admin: End the current voting session
        /// </summary>
        public async Task<string> AdminEndVoting()
        {
            var json = "{}";
            return await PostJsonAsync("/progression/admin/end-voting", json);
        }

        /// <summary>
        /// Admin: Reset the entire progression system
        /// </summary>
        public async Task<string> AdminResetProgression()
        {
            var json = "{}";
            return await PostJsonAsync("/progression/admin/reset", json);
        }

        #endregion

        #region Jobs System

        /// <summary>
        /// Get all available jobs
        /// </summary>
        public async Task<string> GetAllJobs()
        {
            return await GetAsync("/jobs");
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
            return await GetAsync("/jobs/user" + query);
        }

        /// <summary>
        /// Award XP to a user for a specific job (Streamer/Admin only)
        /// </summary>
        public async Task<string> AwardJobXP(string platform, string platformId, string username, string jobName, int xpAmount)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""job_name"":""{3}"",""xp_amount"":{4}}}",
                platform, platformId, username, jobName, xpAmount);
            return await PostJsonAsync("/jobs/award-xp", json);
        }

        #endregion

        #region Account Linking

        /// <summary>
        /// Initiate account linking process
        /// </summary>
        public async Task<string> InitiateLinking(string platform, string platformId, string username)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}""}}",
                platform, platformId, username);
            return await PostJsonAsync("/link/initiate", json);
        }

        /// <summary>
        /// Claim a linking code from another platform
        /// </summary>
        public async Task<string> ClaimLinkingCode(string platform, string platformId, string username, string code)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""code"":""{3}""}}",
                platform, platformId, username, code);
            return await PostJsonAsync("/link/claim", json);
        }

        /// <summary>
        /// Confirm account linking
        /// </summary>
        public async Task<string> ConfirmLinking(string platform, string platformId)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}""}}",
                platform, platformId);
            return await PostJsonAsync("/link/confirm", json);
        }

        /// <summary>
        /// Unlink accounts
        /// </summary>
        public async Task<string> UnlinkAccounts(string platform, string platformId)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}""}}",
                platform, platformId);
            return await PostJsonAsync("/link/unlink", json);
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
            return await GetAsync("/link/status" + query);
        }

        #endregion

        #region Economy (Extended)

        /// <summary>
        /// Get current buy prices for items
        /// </summary>
        public async Task<string> GetBuyPrices()
        {
            return await GetAsync("/prices/buy");
        }

        #endregion

        #region Admin Utilities

        /// <summary>
        /// Reload item name aliases from configuration (Admin only)
        /// </summary>
        public async Task<string> ReloadAliases()
        {
            var json = "{}";
            return await PostJsonAsync("/admin/reload-aliases", json);
        }

        /// <summary>
        /// Test endpoint for debugging
        /// </summary>
        public async Task<string> Test(string platform, string platformId, string username)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}""}}",
                platform, platformId, username);
            return await PostJsonAsync("/test", json);
        }

        #endregion

        #region Message Handler (All-in-One)

        /// <summary>
        /// Handle a chat message (processes commands, tracks engagement, gives rewards)
        /// Use this for Twitch/YouTube chat integration
        /// </summary>
        public async Task<string> HandleMessage(string platform, string platformId, string username, 
            string message, bool isModerator = false, bool isSubscriber = false)
        {
            var json = string.Format(@"{{""platform"":""{0}"",""platform_id"":""{1}"",""username"":""{2}"",""message"":""{3}"",""is_moderator"":{4},""is_subscriber"":{5}}}",
                platform, platformId, username, 
                message.Replace("\"", "\\\"").Replace("\n", "\\n"),
                isModerator.ToString().ToLower(),
                isSubscriber.ToString().ToLower());
            return await PostJsonAsync("/message/handle", json);
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
    /// </summary>
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

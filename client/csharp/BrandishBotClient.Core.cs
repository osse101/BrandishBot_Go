using System.Threading.Tasks;
using System.Collections.Generic;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Register a new user (auto-called on first interaction)
        /// </summary>
        public async Task<User> RegisterUser(string platform, string platformId, string username)
        {
            return await PostAsync<User>("/api/v1/user/register", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Handle a chat message (processes commands, tracks engagement, gives rewards)
        /// </summary>
        public async Task<MessageResult> HandleMessage(string platform, string platformId, string username, string message)
        {
            return await PostAsync<MessageResult>("/api/v1/message/handle", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                message = message
            });
        }

        /// <summary>
        /// Get the backend version
        /// </summary>
        public async Task<VersionInfo> GetVersion()
        {
            return await GetAsync<VersionInfo>("/version");
        }

        /// <summary>
        /// Get feature information for a specific platform
        /// </summary>
        /// <param name="platform">Platform name (discord, twitch, youtube)</param>
        /// <param name="feature">Feature name (optional, defaults to overview)</param>
        /// <param name="topic">Topic name within feature (optional, for hierarchical features)</param>
        public async Task<InfoResponse> GetInfo(string platform, string feature = null, string topic = null)
        {
            var queryParams = new List<string> { $"platform={platform}" };
            
            if (!string.IsNullOrEmpty(feature))
                queryParams.Add($"feature={feature}");
            
            if (!string.IsNullOrEmpty(topic))
                queryParams.Add($"topic={topic}");
            
            string query = string.Join("&", queryParams);
            return await GetAsync<InfoResponse>($"/api/v1/info?{query}");
        }

        /// <summary>
        /// Check if API is alive
        /// </summary>
        public async Task<string> HealthCheck()
        {
            return await GetAsync<string>("/healthz");
        }

        /// <summary>
        /// Check if API is ready
        /// </summary>
        public async Task<string> ReadyCheck()
        {
            return await GetAsync<string>("/readyz");
        }
    }
}

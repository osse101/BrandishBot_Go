using System.Threading.Tasks;

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
        /// Get feature information
        /// </summary>
        public async Task<string> GetInfo()
        {
            return await GetAsync<string>("/api/v1/info");
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

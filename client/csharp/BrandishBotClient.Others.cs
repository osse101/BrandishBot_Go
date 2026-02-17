using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        #region Gamble System

        /// <summary>
        /// Start a new gamble session
        /// </summary>
        public async Task<StartGambleResponse> StartGamble(string platform, string platformId, string username,
            string itemName, int quantity = 1)
        {
            return await PostAsync<StartGambleResponse>("/api/v1/gamble/start", new
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
        public async Task<SuccessResponse> JoinGamble(string gambleId, string platform, string platformId, string username)
        {
            return await PostAsync<SuccessResponse>("/api/v1/gamble/join?id=" + gambleId, new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Get active gamble details
        /// </summary>
        public async Task<Gamble> GetActiveGamble()
        {
            return await GetAsync<Gamble>("/api/v1/gamble/active");
        }

        /// <summary>
        /// Get gamble details by ID
        /// </summary>
        public async Task<Gamble> GetGamble(string gambleId)
        {
            return await GetAsync<Gamble>("/api/v1/gamble/get?id=" + gambleId);
        }

        #endregion
    }
}

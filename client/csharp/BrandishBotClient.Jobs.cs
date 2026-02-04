using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
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
            return await GetAsync<string>("/api/v1/jobs/user" + query);
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
            return await GetAsync<string>("/api/v1/jobs/user" + query);
        }

        /// <summary>
        /// Award XP to a user for a specific job (Streamer/Admin only)
        /// </summary>
        public async Task<SuccessResponse> AwardJobXP(string platform, string username, string jobKey, int xpAmount)
        {
            return await PostAsync<SuccessResponse>("/api/v1/jobs/award-xp", new
            {
                platform = platform,
                username = username,
                job_key = jobKey,
                xp_amount = xpAmount
            });
        }
    }
}

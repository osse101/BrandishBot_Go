using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Get user statistics (self-mode)
        /// </summary>
        public async Task<StatsSummary> GetUserStats(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync<StatsSummary>("/api/v1/stats/user" + query);
        }

        /// <summary>
        /// Get user statistics by username (target-mode)
        /// </summary>
        public async Task<StatsSummary> GetUserStatsByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync<StatsSummary>("/api/v1/stats/user" + query);
        }

        /// <summary>
        /// Get system-wide statistics
        /// </summary>
        public async Task<StatsSummary> GetSystemStats()
        {
            return await GetAsync<StatsSummary>("/api/v1/stats/system");
        }

        /// <summary>
        /// Get leaderboard
        /// </summary>
        public async Task<List<LeaderboardEntry>> GetLeaderboard(string eventType = "engagement_score", string period = "daily", int limit = 10)
        {
            var query = BuildQuery(
                "event_type=" + eventType,
                "period=" + period,
                "limit=" + limit.ToString()
            );
            return await GetAsync<List<LeaderboardEntry>>("/api/v1/stats/leaderboard" + query);
        }

        /// <summary>
        /// Get user timeout status
        /// </summary>
        public async Task<string> GetUserTimeout(string platform, string username)
        {
            var query = BuildQuery("platform=" + platform, "username=" + username);
            return await GetAsync<string>("/api/v1/user/timeout" + query);
        }
    }
}

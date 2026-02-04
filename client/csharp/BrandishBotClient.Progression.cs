using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Get progression tree structure
        /// </summary>
        public async Task<string> GetProgressionTree()
        {
            return await GetAsync<string>("/api/v1/progression/tree");
        }

        /// <summary>
        /// Get available (unlockable) progression nodes
        /// </summary>
        public async Task<string> GetAvailableNodes()
        {
            return await GetAsync<string>("/api/v1/progression/available");
        }

        /// <summary>
        /// Vote to unlock a progression node
        /// </summary>
        public async Task<SuccessResponse> VoteForNode(string platform, string platformId, string nodeKey)
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/vote", new
            {
                platform = platform,
                platform_id = platformId,
                node_key = nodeKey
            });
        }

        /// <summary>
        /// Get progression status (unlocked nodes, votes, etc.)
        /// </summary>
        public async Task<ProgressionStatus> GetProgressionStatus()
        {
            return await GetAsync<ProgressionStatus>("/api/v1/progression/status");
        }

        /// <summary>
        /// Get user engagement breakdown (contribution points)
        /// </summary>
        public async Task<string> GetUserEngagement(string platform, string platformId)
        {
            var query = BuildQuery("platform=" + platform, "platform_id=" + platformId);
            return await GetAsync<string>("/api/v1/progression/engagement" + query);
        }

        /// <summary>
        /// Get user engagement breakdown by username (contribution points)
        /// </summary>
        public async Task<string> GetUserEngagementByUsername(string platform, string username)
        {
            var query = BuildQuery("platform=" + platform, "username=" + username);
            return await GetAsync<string>("/api/v1/progression/engagement-by-username" + query);
        }

        /// <summary>
        /// Get current voting session details
        /// </summary>
        public async Task<VotingSession> GetVotingSession()
        {
            return await GetAsync<VotingSession>("/api/v1/progression/session");
        }

        /// <summary>
        /// Get unlock progress for the current voting session
        /// </summary>
        public async Task<UnlockProgress> GetUnlockProgress()
        {
            return await GetAsync<UnlockProgress>("/api/v1/progression/unlock-progress");
        }
    }
}

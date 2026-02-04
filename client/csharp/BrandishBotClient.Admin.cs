using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Admin: Unlock a specific node at a specific level
        /// </summary>
        public async Task<SuccessResponse> AdminUnlockNode(string nodeKey, int level = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/unlock", new { node_key = nodeKey, level = level });
        }

        /// <summary>
        /// Admin: Unlock ALL progression nodes at max level (DEBUG ONLY)
        /// </summary>
        public async Task<SuccessResponse> AdminUnlockAllNodes()
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/unlock-all", new { });
        }

        /// <summary>
        /// Admin: Re-lock a specific node at a specific level
        /// </summary>
        public async Task<SuccessResponse> AdminRelockNode(string nodeKey, int level)
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/relock", new { node_key = nodeKey, level = level });
        }

        /// <summary>
        /// Admin: Start a new voting session
        /// </summary>
        public async Task<SuccessResponse> AdminStartVoting()
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/start-voting", new { });
        }

        /// <summary>
        /// Admin: End the current voting session
        /// </summary>
        public async Task<SuccessResponse> AdminEndVoting()
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/end-voting", new { });
        }

        /// <summary>
        /// Admin: Reset the entire progression system
        /// </summary>
        public async Task<SuccessResponse> AdminResetProgression(string resetBy, string reason, bool preserveUserProgression)
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/reset", new 
            { 
                reset_by = resetBy,
                reason = reason,
                preserve_user_progression = preserveUserProgression
            });
        }

        /// <summary>
        /// Admin: Add contribution points to the progression system
        /// </summary>
        public async Task<SuccessResponse> AdminAddContribution(int amount)
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/contribution", new { amount = amount });
        }

        /// <summary>
        /// Admin: Reload engagement weight cache
        /// </summary>
        public async Task<SuccessResponse> AdminReloadWeights()
        {
            return await PostAsync<SuccessResponse>("/api/v1/admin/progression/reload-weights", new { });
        }

        /// <summary>
        /// Admin: Get user cache statistics
        /// </summary>
        public async Task<string> AdminGetCacheStats()
        {
            return await GetAsync<string>("/api/v1/admin/cache/stats");
        }

        /// <summary>
        /// Admin: Clear a user's timeout
        /// </summary>
        public async Task<SuccessResponse> AdminClearTimeoutAsync(string platform, string username)
        {
            return await PostAsync<SuccessResponse>("/api/v1/admin/timeout/clear", new
            {
                platform = platform,
                username = username
            });
        }
    }
}

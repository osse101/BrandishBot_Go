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
        /// <summary>
        /// Admin: Force end the current voting session immediately
        /// </summary>
        public async Task<SuccessResponse> AdminForceEndVoting()
        {
            return await PostAsync<SuccessResponse>("/api/v1/progression/admin/force-end-voting", new { });
        }

        /// <summary>
        /// Admin: Instantly unlock a node (bypassing vote)
        /// </summary>
        public async Task<SuccessResponse> AdminInstantUnlock(string nodeKey)
        {
             return await PostAsync<SuccessResponse>("/api/v1/progression/admin/instant-unlock", new { node_key = nodeKey });
        }

        #region User Management

        /// <summary>
        /// Admin: Lookup user details (including inventory and stats)
        /// </summary>
        public async Task<AdminUserLookupResponse> AdminUserLookup(string platform, string username)
        {
            var query = BuildQuery("platform=" + platform, "username=" + username);
            return await GetAsync<AdminUserLookupResponse>("/api/v1/admin/users/lookup" + query);
        }

        /// <summary>
        /// Admin: Get recent users
        /// </summary>
        public async Task<List<User>> AdminGetRecentUsers(int limit = 10)
        {
            return await GetAsync<List<User>>("/api/v1/admin/users/recent?limit=" + limit);
        }

        /// <summary>
        /// Admin: Get active chatters
        /// </summary>
        public async Task<List<User>> AdminGetActiveChatters(int minutes = 10)
        {
            return await GetAsync<List<User>>("/api/v1/admin/users/active?minutes=" + minutes);
        }

        #endregion

        #region Autocomplete & Lists

        /// <summary>
        /// Admin: Get all items (for autocomplete)
        /// </summary>
        public async Task<List<string>> AdminGetItems()
        {
            return await GetAsync<List<string>>("/api/v1/admin/items");
        }

        /// <summary>
        /// Admin: Get all jobs (for autocomplete)
        /// </summary>
        public async Task<List<string>> AdminGetJobs()
        {
            return await GetAsync<List<string>>("/api/v1/admin/jobs");
        }

        #endregion

        #region Events

        /// <summary>
        /// Admin: Get event logs
        /// </summary>
        public async Task<List<string>> AdminGetEvents(int limit = 50)
        {
            return await GetAsync<List<string>>("/api/v1/admin/events?limit=" + limit);
        }

        /// <summary>
        /// Admin: Reload notification aliases
        /// </summary>
        public async Task<SuccessResponse> AdminReloadAliases()
        {
             return await PostAsync<SuccessResponse>("/api/v1/admin/reload-aliases", new { });
        }

        #endregion

        #region Jobs

        /// <summary>
        /// Admin: Award Job XP to a user
        /// </summary>
        public async Task<SuccessResponse> AdminAwardJobXP(string platform, string username, string jobKey, int amount)
        {
            return await PostAsync<SuccessResponse>("/api/v1/admin/jobs/award-xp", new
            {
                platform = platform,
                username = username,
                job_key = jobKey,
                amount = amount
            });
        }

        /// <summary>
        /// Admin: Manually trigger daily reset
        /// </summary>
        public async Task<SuccessResponse> AdminManualDailyReset()
        {
            return await PostAsync<SuccessResponse>("/api/v1/admin/jobs/reset-daily-xp", new { });
        }

        /// <summary>
        /// Admin: Get daily reset status
        /// </summary>
        public async Task<ResetStatusResponse> AdminGetResetStatus()
        {
            return await GetAsync<ResetStatusResponse>("/api/v1/admin/jobs/reset-status");
        }

        #endregion

        #region System & Metrics

        /// <summary>
        /// Admin: Get system metrics
        /// </summary>
        public async Task<string> AdminGetMetrics()
        {
            return await GetAsync<string>("/api/v1/admin/metrics");
        }

        /// <summary>
        /// Admin: Broadcast custom SSE event
        /// </summary>
        public async Task<SuccessResponse> AdminBroadcastSSE(string eventType, object payload)
        {
            return await PostAsync<SuccessResponse>("/api/v1/admin/sse/broadcast", new
            {
                event_type = eventType,
                payload = payload
            });
        }

        #endregion

        #region Simulation

        /// <summary>
        /// Admin: Get scenario capabilities
        /// </summary>
        public async Task<List<ScenarioCapability>> AdminGetScenarioCapabilities()
        {
             return await GetAsync<List<ScenarioCapability>>("/api/v1/admin/simulate/capabilities");
        }

        /// <summary>
        /// Admin: Get available scenarios
        /// </summary>
        public async Task<List<ScenarioDefinition>> AdminGetScenarios()
        {
             return await GetAsync<List<ScenarioDefinition>>("/api/v1/admin/simulate/scenarios");
        }

        /// <summary>
        /// Admin: Get specific scenario definition
        /// </summary>
        public async Task<ScenarioDefinition> AdminGetScenario(string scenarioId)
        {
             return await GetAsync<ScenarioDefinition>("/api/v1/admin/simulate/scenario?id=" + scenarioId);
        }

        /// <summary>
        /// Admin: Run a predefined scenario
        /// </summary>
        public async Task<ScenarioRunResult> AdminRunScenario(string scenarioId, Dictionary<string, string> parameters = null)
        {
            return await PostAsync<ScenarioRunResult>("/api/v1/admin/simulate/run", new
            {
                scenario_id = scenarioId,
                parameters = parameters ?? new Dictionary<string, string>()
            });
        }

        /// <summary>
        /// Admin: Run a custom scenario
        /// </summary>
        public async Task<ScenarioRunResult> AdminRunCustomScenario(string name, string stepsYaml)
        {
            return await PostAsync<ScenarioRunResult>("/api/v1/admin/simulate/run-custom", new
            {
                name = name,
                steps = stepsYaml
            });
        }

        #endregion
    }
}

using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Start a new expedition
        /// </summary>
        public async Task<StartExpeditionResponse> StartExpedition(string platform, string platformId, string username, string expeditionType = "standard")
        {
            return await PostAsync<StartExpeditionResponse>("/api/v1/expedition/start", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                expedition_type = expeditionType
            });
        }

        /// <summary>
        /// Join an active expedition
        /// </summary>
        public async Task<SuccessResponse> JoinExpedition(string platform, string platformId, string username, string expeditionId)
        {
            return await PostAsync<SuccessResponse>("/api/v1/expedition/join?id=" + expeditionId, new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Get expedition details by ID
        /// </summary>
        public async Task<ExpeditionDetails> GetExpedition(string expeditionId)
        {
            return await GetAsync<ExpeditionDetails>("/api/v1/expedition/get?id=" + expeditionId);
        }

        /// <summary>
        /// Get the currently active expedition
        /// </summary>
        public async Task<ExpeditionDetails> GetActiveExpedition()
        {
            return await GetAsync<ExpeditionDetails>("/api/v1/expedition/active");
        }

        /// <summary>
        /// Get the current expedition system status (active expedition + cooldown)
        /// </summary>
        public async Task<ExpeditionStatus> GetExpeditionStatus()
        {
            return await GetAsync<ExpeditionStatus>("/api/v1/expedition/status");
        }

        /// <summary>
        /// Get the journal entries for a completed expedition
        /// </summary>
        public async Task<List<ExpeditionJournalEntry>> GetExpeditionJournal(string expeditionId)
        {
            return await GetAsync<List<ExpeditionJournalEntry>>("/api/v1/expedition/journal?id=" + expeditionId);
        }
    }
}

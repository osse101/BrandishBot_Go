using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        #region Quests

        /// <summary>
        /// Get all active quests
        /// </summary>
        public async Task<List<Quest>> GetActiveQuests()
        {
            var response = await GetAsync<QuestListResponse>("/api/v1/quests/active");
            return response?.Quests ?? new List<Quest>();
        }

        /// <summary>
        /// Get user's progress on active quests
        /// </summary>
        public async Task<List<UserQuestProgress>> GetUserQuestProgress(string platform, string platformId, string username)
        {
             var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId,
                "username=" + username
            );
            return await GetAsync<List<UserQuestProgress>>("/api/v1/quests/progress" + query);
        }

        /// <summary>
        /// Claim a quest reward
        /// </summary>
        public async Task<SuccessResponse> ClaimQuestReward(string platform, string platformId, string username, string questKey)
        {
            return await PostAsync<SuccessResponse>("/api/v1/quests/claim", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                quest_key = questKey
            });
        }

        #endregion
    }
}

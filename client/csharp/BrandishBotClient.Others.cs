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
            return await GetAsync<Gamble>("/api/v1/gamble/get");
        }

        #endregion

        #region Account Linking

        /// <summary>
        /// Initiate account linking process
        /// </summary>
        public async Task<SuccessResponse> InitiateLinking(string platform, string platformId, string username)
        {
            return await PostAsync<SuccessResponse>("/api/v1/link/initiate", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Claim a linking code from another platform
        /// </summary>
        public async Task<SuccessResponse> ClaimLinkingCode(string platform, string platformId, string username, string code)
        {
            return await PostAsync<SuccessResponse>("/api/v1/link/claim", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                code = code
            });
        }

        /// <summary>
        /// Confirm account linking
        /// </summary>
        public async Task<SuccessResponse> ConfirmLinking(string platform, string platformId)
        {
            return await PostAsync<SuccessResponse>("/api/v1/link/confirm", new
            {
                platform = platform,
                platform_id = platformId
            });
        }

        /// <summary>
        /// Unlink accounts
        /// </summary>
        public async Task<SuccessResponse> UnlinkAccounts(string platform, string platformId, string targetPlatform)
        {
            return await PostAsync<SuccessResponse>("/api/v1/link/unlink", new
            {
                platform = platform,
                platform_id = platformId,
                target_platform = targetPlatform
            });
        }

        /// <summary>
        /// Get linking status for a user
        /// </summary>
        public async Task<LinkingStatus> GetLinkingStatus(string platform, string platformId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId
            );
            return await GetAsync<LinkingStatus>("/api/v1/link/status" + query);
        }

        #endregion

        #region Predictions

        /// <summary>
        /// Process a prediction outcome from Twitch/YouTube
        /// </summary>
        public async Task<PredictionResult> ProcessPredictionOutcome(
            string platform,
            PredictionWinner winner,
            int totalPointsSpent,
            List<PredictionParticipant> participants)
        {
            return await PostAsync<PredictionResult>("/api/v1/prediction", new
            {
                platform = platform,
                winner = winner,
                total_points_spent = totalPointsSpent,
                participants = participants
            });
        }

        #endregion
    }
}

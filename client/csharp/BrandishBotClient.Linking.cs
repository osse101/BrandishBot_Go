using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        #region Account Linking

        /// <summary>
        /// Initiate account linking process
        /// </summary>
        public async Task<LinkInitiateResponse> InitiateLinking(string platform, string platformId, string username)
        {
            return await PostAsync<LinkInitiateResponse>("/api/v1/link/initiate", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Claim a linking code from another platform
        /// </summary>
        public async Task<LinkClaimResponse> ClaimLinkingCode(string platform, string platformId, string username, string code)
        {
            return await PostAsync<LinkClaimResponse>("/api/v1/link/claim", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                token = code
            });
        }

        /// <summary>
        /// Confirm account linking
        /// </summary>
        public async Task<LinkConfirmResponse> ConfirmLinking(string platform, string platformId)
        {
            return await PostAsync<LinkConfirmResponse>("/api/v1/link/confirm", new
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
    }
}

using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Deposit items into the user's compost bin.
        /// Auto-starts composting on first deposit. Additional deposits extend the timer.
        /// Items must have the "compostable" tag. Cannot deposit when bin is ready/sludge.
        /// </summary>
        public async Task<CompostDepositResponse> CompostDeposit(string platform, string platformId, List<CompostDepositItem> items)
        {
            return await PostAsync<CompostDepositResponse>("/api/v1/compost/deposit", new
            {
                platform = platform,
                platform_id = platformId,
                items = items
            });
        }

        /// <summary>
        /// Harvest items from the compost bin, or check status if not ready.
        /// Returns harvest output if composting is complete, or current status/timers if still composting.
        /// </summary>
        public async Task<CompostHarvestResponse> CompostHarvest(string platform, string platformId, string username)
        {
            return await PostAsync<CompostHarvestResponse>("/api/v1/compost/harvest", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Check the current compost bin status without triggering a harvest.
        /// Returns full status including item list, timers, and capacity.
        /// </summary>
        public async Task<CompostStatusResponse> CompostStatus(string platform, string platformId)
        {
            var query = "?platform=" + System.Uri.EscapeDataString(platform)
                      + "&platform_id=" + System.Uri.EscapeDataString(platformId);
            return await GetAsync<CompostStatusResponse>("/api/v1/compost/status" + query);
        }
    }
}

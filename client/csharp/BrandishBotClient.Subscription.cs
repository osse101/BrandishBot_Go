using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Send a subscription event to BrandishBot (called by Streamer.bot on subscription events)
        /// </summary>
        /// <param name="evt">Subscription event details</param>
        /// <returns>True if the event was successfully processed</returns>
        public async Task<bool> SendSubscriptionEventAsync(SubscriptionEvent evt)
        {
            var response = await PostAsync<SuccessResponse>("/api/v1/subscriptions/event", evt);
            return response != null;
        }

        /// <summary>
        /// Get a user's subscription status by platform and platform user ID
        /// </summary>
        /// <param name="platform">Platform (twitch or youtube)</param>
        /// <param name="platformUserId">Platform-specific user ID</param>
        /// <returns>Subscription details including tier information</returns>
        public async Task<SubscriptionWithTier> GetUserSubscriptionAsync(string platform, string platformUserId)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformUserId
            );
            return await GetAsync<SubscriptionWithTier>("/api/v1/subscriptions/user" + query);
        }
    }
}

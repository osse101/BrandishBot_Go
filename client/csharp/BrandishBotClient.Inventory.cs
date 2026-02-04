using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Get user's inventory
        /// </summary>
        public async Task<GetInventoryResponse> GetInventory(string platform, string platformId, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "platform_id=" + platformId,
                "username=" + username
            );
            return await GetAsync<GetInventoryResponse>("/api/v1/user/inventory" + query);
        }

        /// <summary>
        /// Get user's inventory by username (no platformId required)
        /// </summary>
        public async Task<GetInventoryResponse> GetInventoryByUsername(string platform, string username)
        {
            var query = BuildQuery(
                "platform=" + platform,
                "username=" + username
            );
            return await GetAsync<GetInventoryResponse>("/api/v1/user/inventory-by-username" + query);
        }

        /// <summary>
        /// Add item by username
        /// </summary>
        public async Task<SuccessResponse> AddItemByUsername(string platform, string username, string itemName, int quantity = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/add", new AddItemByUsernameRequest
            {
                Platform = platform,
                Username = username,
                ItemName = itemName,
                Quantity = quantity
            });
        }
        
        /// <summary>
        /// Remove item by username
        /// </summary>
        public async Task<SuccessResponse> RemoveItemByUsername(string platform, string username, string itemName, int quantity = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/remove", new
            {
                platform = platform,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Give item from one user to another
        /// </summary>
        public async Task<SuccessResponse> GiveItem(string fromPlatform, string fromPlatformId, string fromUsername, 
            string toPlatform, string toUsername, string itemName, int quantity = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/give", new
            {
                from_platform = fromPlatform,
                from_platform_id = fromPlatformId,
                from_username = fromUsername,
                to_platform = toPlatform,
                to_username = toUsername,
                item_name = itemName,
                quantity = quantity
            });
        }
    }
}

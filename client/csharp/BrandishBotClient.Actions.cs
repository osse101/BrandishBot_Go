using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Use an item (opens lootboxes, activates items, etc.)
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="platformId">The user's platform ID</param>
        /// <param name="username">The user's username</param>
        /// <param name="itemName">The item to use</param>
        /// <param name="quantity">Quantity to use (default 1)</param>
        /// <param name="targetUsername">Optional target username</param>
        public async Task<SuccessResponse> UseItem(string platform, string platformId, string username, 
            string itemName, int quantity = 1, string targetUsername = null)
        {
            var data = new Dictionary<string, object>
            {
                { "platform", platform },
                { "platform_id", platformId },
                { "username", username },
                { "item_name", itemName },
                { "quantity", quantity }
            };

            if (!string.IsNullOrEmpty(targetUsername))
            {
                data["target_user"] = targetUsername;
            }

            return await PostAsync<SuccessResponse>("/api/v1/user/item/use", data);
        }

        /// <summary>
        /// Search for items (opens random lootboxes based on engagement)
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="platformId">The user's platform ID</param>
        /// <param name="username">The user's username</param>
        public async Task<SuccessResponse> Search(string platform, string platformId, string username)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/search", new
            {
                platform = platform,
                platform_id = platformId,
                username = username
            });
        }

        /// <summary>
        /// Upgrade an item using a recipe
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="platformId">The user's platform ID</param>
        /// <param name="username">The user's username</param>
        /// <param name="itemName">The item to upgrade</param>
        /// <param name="quantity">Quantity to upgrade (default 1)</param>
        public async Task<SuccessResponse> UpgradeItem(string platform, string platformId, string username, string itemName, int quantity = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/upgrade", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Disassemble an item to get materials
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="platformId">The user's platform ID</param>
        /// <param name="username">The user's username</param>
        /// <param name="itemName">The item to disassemble</param>
        /// <param name="quantity">Quantity to disassemble (default 1)</param>
        public async Task<SuccessResponse> DisassembleItem(string platform, string platformId, string username, string itemName, int quantity = 1)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/disassemble", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Get recipes unlocked by the user
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="platformId">The user's platform ID</param>
        /// <param name="username">The user's username</param>
        public async Task<List<Recipe>> GetUnlockedRecipes(string platform, string platformId, string username)
        {
            var query = BuildQuery("platform=" + System.Uri.EscapeDataString(platform), "platform_id=" + System.Uri.EscapeDataString(platformId), "user=" + System.Uri.EscapeDataString(username));
            var response = await GetAsync<RecipeListResponse>("/api/v1/recipes" + query);
            return response?.Recipes ?? new List<Recipe>();
        }

        /// <summary>
        /// Get recipes unlocked by the user by username
        /// </summary>
        /// <param name="platform">The platform (e.g. "twitch", "discord")</param>
        /// <param name="username">The user's username</param>
        public async Task<List<Recipe>> GetUnlockedRecipesByUsername(string platform, string username)
        {
            var query = BuildQuery("platform=" + System.Uri.EscapeDataString(platform), "user=" + System.Uri.EscapeDataString(username));
            var response = await GetAsync<RecipeListResponse>("/api/v1/recipes" + query);
            return response?.Recipes ?? new List<Recipe>();
        }
    }
}

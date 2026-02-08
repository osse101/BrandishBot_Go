using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        /// <summary>
        /// Buy an item from the shop
        /// </summary>
        public async Task<SuccessResponse> BuyItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/buy", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Sell an item from inventory
        /// </summary>
        public async Task<SuccessResponse> SellItem(string platform, string platformId, string username, string itemName, int quantity)
        {
            return await PostAsync<SuccessResponse>("/api/v1/user/item/sell", new
            {
                platform = platform,
                platform_id = platformId,
                username = username,
                item_name = itemName,
                quantity = quantity
            });
        }

        /// <summary>
        /// Get current buy prices for items
        /// </summary>
        public async Task<List<Item>> GetBuyPrices()
        {
            return await GetAsync<List<Item>>("/api/v1/prices/buy");
        }

        /// <summary>
        /// Get current item prices (Sell Prices)
        /// </summary>
        public async Task<List<Item>> GetSellPrices()
        {
            return await GetAsync<List<Item>>("/api/v1/prices");
        }

        /// <summary>
        /// Get available crafting recipes
        /// </summary>
        public async Task<List<Recipe>> GetRecipes()
        {
            var response = await GetAsync<RecipeListResponse>("/api/v1/recipes");
            return response?.Recipes ?? new List<Recipe>();
        }
    }
}

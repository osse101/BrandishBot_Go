using System;
using System.Threading.Tasks;
using BrandishBot.Client;

/// <summary>
/// Streamer.bot wrapper for BrandishBotClient
/// This class manages the singleton and provides public bool methods for CPH.runMethod()
/// 
/// SETUP:
/// 1. Update baseUrl and apiKey constants below
/// 2. Add this file to your streamer.bot action as C# code
/// 3. Call methods using: CPH.runMethod("YourActionName", "MethodName")
/// </summary>
public class CPHInline
{
    // Singleton client - shared across all method calls in this action
    private static BrandishBotClient client;

    // Initialize the client (called automatically on first use)
    private void EnsureInitialized()
    {
        if (client == null)
        {
            string baseUrl = CPH.GetGlobalVar<string>("ServerBaseURL", persisted:true);
            string apiKey = CPH.GetGlobalVar<string>("ServerApiKey", persisted:true);
            
            if (string.IsNullOrEmpty(baseUrl))
            {
                CPH.LogError("CONFIGURATION ERROR: ServerBaseURL global variable is not set!");
                CPH.LogError("Name: ServerBaseURL, Value: http://IP:PORT (or your server URL)");
                throw new InvalidOperationException("ServerBaseURL not configured");
            }
            
            if (string.IsNullOrEmpty(apiKey))
            {
                CPH.LogError("CONFIGURATION ERROR: ServerApiKey global variable is not set!");
                CPH.LogError("Name: ServerApiKey, Value: your-api-key-here");
                throw new InvalidOperationException("ServerApiKey not configured");
            }
            
            BrandishBotClient.Initialize(baseUrl, apiKey);
            client = BrandishBotClient.Instance;
        }
    }

    #region User Management

    /// <summary>
    /// Register a new user
    /// Uses: userType, userId, userName (from streamer.bot context)
    /// </summary>
    public bool RegisterUser()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;

        try
        {
            var result = client.RegisterUser(platform, platformId, username).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"RegisterUser failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Inventory Operations

    /// <summary>
    /// Get user's inventory
    /// Uses: userType, userId (from streamer.bot context)
    /// </summary>
    public bool GetInventory()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;

        try
        {
            var result = client.GetInventory(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetInventory failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Add item to user's inventory (Admin/Streamer only)
    /// Uses: userType, userId (from streamer.bot)
    /// Args: item_id, quantity
    /// </summary>
    public bool AddItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.AddItem(platform, platformId, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"AddItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Remove item from user's inventory (Admin/Streamer only)
    /// Uses: userType, userId (from streamer.bot)
    /// Args: item_id, quantity
    /// </summary>
    public bool RemoveItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.RemoveItem(platform, platformId, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"RemoveItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Give item from one user to another
    /// Args: from_platform, from_platform_id, to_platform, to_platform_id, to_username, item_id, quantity
    /// </summary>
    public bool GiveItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("from_platform", out string fromPlatform)) return false;
        if (!CPH.TryGetArg("from_platform_id", out string fromPlatformId)) return false;
        if (!CPH.TryGetArg("to_platform", out string toPlatform)) return false;
        if (!CPH.TryGetArg("to_platform_id", out string toPlatformId)) return false;
        if (!CPH.TryGetArg("to_username", out string toUsername)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.GiveItem(fromPlatform, fromPlatformId, toPlatform, toPlatformId, toUsername, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GiveItem failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Economy

    /// <summary>
    /// Buy an item from the shop
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: item_id, quantity
    /// </summary>
    public bool BuyItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.BuyItem(platform, platformId, username, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"BuyItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Sell an item from inventory
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: item_id, quantity
    /// </summary>
    public bool SellItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.SellItem(platform, platformId, username, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"SellItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Get current item prices
    /// Args: (none)
    /// </summary>
    public bool GetPrices()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetPrices().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetPrices failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Item Actions

    /// <summary>
    /// Use an item (opens lootboxes, activates items, etc.)
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: item_id, quantity, target_username (optional)
    /// </summary>
    public bool UseItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;
        
        CPH.TryGetArg("target_username", out string targetUsername);

        try
        {
            var result = client.UseItem(platform, platformId, username, itemId, quantity, targetUsername).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"UseItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Search for items (opens random lootboxes based on engagement)
    /// Uses: userType, userId, userName (from streamer.bot context)
    /// </summary>
    public bool Search()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;

        try
        {
            var result = client.Search(platform, platformId, username).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"Search failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Crafting

    /// <summary>
    /// Upgrade an item using a recipe
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: recipe_id
    /// </summary>
    public bool UpgradeItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("recipe_id", out int recipeId)) return false;

        try
        {
            var result = client.UpgradeItem(platform, platformId, username, recipeId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"UpgradeItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Disassemble an item to get materials
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: item_id, quantity
    /// </summary>
    public bool DisassembleItem()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("item_id", out int itemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.DisassembleItem(platform, platformId, username, itemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"DisassembleItem failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Get available crafting recipes
    /// Args: (none)
    /// </summary>
    public bool GetRecipes()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetRecipes().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetRecipes failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Gamble System

    /// <summary>
    /// Start a new gamble session
    /// Uses: userType, userId, userName (from streamer.bot)
    /// Args: lootbox_item_id, quantity
    /// </summary>
    public bool StartGamble()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("lootbox_item_id", out int lootboxItemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.StartGamble(platform, platformId, username, lootboxItemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"StartGamble failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Join an existing gamble session
    /// Args: gamble_id, platform, platform_id, username, lootbox_item_id, quantity
    /// </summary>
    public bool JoinGamble()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("gamble_id", out string gambleId)) return false;
        if (!CPH.TryGetArg("platform", out string platform)) return false;
        if (!CPH.TryGetArg("platform_id", out string platformId)) return false;
        if (!CPH.TryGetArg("username", out string username)) return false;
        if (!CPH.TryGetArg("lootbox_item_id", out int lootboxItemId)) return false;
        if (!CPH.TryGetArg("quantity", out int quantity)) return false;

        try
        {
            var result = client.JoinGamble(gambleId, platform, platformId, username, lootboxItemId, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"JoinGamble failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Get active gamble details
    /// Args: (none)
    /// </summary>
    public bool GetActiveGamble()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetActiveGamble().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetActiveGamble failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Message Handler

    /// <summary>
    /// Handle a chat message (processes commands, tracks engagement, gives rewards)
    /// Uses: userType, userId, userName, message (from streamer.bot)
    /// </summary>
    public bool HandleMessage()
    {
        EnsureInitialized();
        
        if (!CPH.TryGetArg("userType", out string platform)) return false;
        if (!CPH.TryGetArg("userId", out string platformId)) return false;
        if (!CPH.TryGetArg("userName", out string username)) return false;
        if (!CPH.TryGetArg("message", out string message)) return false;

        try
        {
            var result = client.HandleMessage(platform, platformId, username, message).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (AggregateException aex)
        {
            var innerEx = aex.InnerException ?? aex;
            CPH.LogError($"HandleMessage failed: {innerEx.GetType().Name}: {innerEx.Message}");
            if (innerEx.InnerException != null)
            {
                CPH.LogError($"Inner exception: {innerEx.InnerException.Message}");
            }
            return false;
        }
        catch (Exception ex)
        {
            CPH.LogError($"HandleMessage failed: {ex.GetType().Name}: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Health Checks

    /// <summary>
    /// Check if API is alive
    /// Args: (none)
    /// </summary>
    public bool HealthCheck()
    {
        EnsureInitialized();

        try
        {
            var result = client.HealthCheck().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"HealthCheck failed: {ex.Message}");
            return false;
        }
    }

    #endregion
}

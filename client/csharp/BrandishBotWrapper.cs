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
            //string baseUrl = "http://127.0.0.1:8080";
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

    /// <summary>
    /// Helper: Validate context arguments (userType, userId, userName)
    /// </summary>
    private bool ValidateContext(out string platform, out string platformId, out string username, ref string error)
    {
        platform = null;
        platformId = null;
        username = null;

        if (!CPH.TryGetArg("userType", out platform))
        {
            error = "Context Error: Missing 'userType'.";
            return false;
        }
        if (!CPH.TryGetArg("userId", out platformId))
        {
            error = "Context Error: Missing 'userId'.";
            return false;
        }
        // userName is often useful for logging or display even if not strictly required by some ID-based endpoints
        CPH.TryGetArg("userName", out username);
        
        return true;
    }

    /// <summary>
    /// Helper: Get a string argument from inputX
    /// </summary>
    private bool GetInputString(int index, string paramName, bool required, out string value, ref string error)
    {
        value = null;
        string key = $"input{index}";
        bool exists = CPH.TryGetArg(key, out string inputVal);
        
        if (exists && !string.IsNullOrWhiteSpace(inputVal))
        {
            value = inputVal;
            return true;
        }

        if (required)
        {
            error = $"Missing required argument: <{paramName}>.";
            return false;
        }

        return true;
    }

    /// <summary>
    /// Helper: Get an integer argument from inputX with default value
    /// </summary>
    private bool GetInputInt(int index, string paramName, int defaultValue, out int value, ref string error)
    {
        value = defaultValue;
        string key = $"input{index}";
        bool exists = CPH.TryGetArg(key, out string inputVal);

        if (exists && !string.IsNullOrWhiteSpace(inputVal))
        {
            if (int.TryParse(inputVal, out int parsed))
            {
                value = parsed;
                return true;
            }
            error = $"Invalid argument <{paramName}>: '{inputVal}' is not a number.";
            return false;
        }

        // Return true (success) using default value if not provided
        return true;
    }

    /// <summary>
    /// Get the backend version
    /// Args: (none)
    /// </summary>
    public bool GetVersion()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetVersion().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetVersion failed: {ex.Message}");
            return false;
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
    /// Command: !addItem <item_name> [quantity]
    /// </summary>
    public bool AddItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"AddItem Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !addItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.AddItem(platform, platformId, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AddItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Remove item from user's inventory (Admin/Streamer only)
    /// Command: !removeItem <item_name> [quantity]
    /// </summary>
    public bool RemoveItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"RemoveItem Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !removeItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.RemoveItem(platform, platformId, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"RemoveItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Give item from one user to another
    /// Command: !giveItem <target_user> <item_name> [quantity]
    /// </summary>
    public bool GiveItem()
    {
        EnsureInitialized();
        
        // This command is triggered by the Sender. 
        // We need: Sender (context), Receiver (target_username arg), Item, Quantity
        string error = null;
        if (!ValidateContext(out string fromPlatform, out string fromPlatformId, out string fromUsername, ref error))
        {
             CPH.LogWarn($"GiveItem Context Error: {error}");
             return false;
        }

        if (!GetInputString(0, "target_user", true, out string toUsername, ref error) ||
            !GetInputString(1, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(2, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !giveItem <target_user> <item_name> [quantity]");
            return true;
        }

        string toPlatform = fromPlatform; 
        string toPlatformId = ""; // Unknown ID from just a username input

        try
        {
            var result = client.GiveItem(fromPlatform, fromPlatformId, toPlatform, toPlatformId, toUsername, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GiveItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Economy

    /// <summary>
    /// Buy an item from the shop
    /// Command: !buyItem <item_name> [quantity]
    /// </summary>
    public bool BuyItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"BuyItem Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !buyItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.BuyItem(platform, platformId, username, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"BuyItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Sell an item from inventory
    /// Command: !sellItem <item_name> [quantity]
    /// </summary>
    public bool SellItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"SellItem Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !sellItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.SellItem(platform, platformId, username, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"SellItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get current item sell prices (Alias for GetPrices)
    /// Args: (none)
    /// </summary>
    public bool GetSellPrices()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetSellPrices().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetSellPrices failed: {ex.Message}");
            return false;
        }
    }

    /// <summary>
    /// Get current item buy prices
    /// Args: (none)
    /// </summary>
    public bool GetBuyPrices()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetBuyPrices().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"GetBuyPrices failed: {ex.Message}");
            return false;
        }
    }

    #endregion

    #region Item Actions

    /// <summary>
    /// Use an item (opens lootboxes, activates items, etc.)
    /// Command: !useItem <item_name> [quantity] [target_user]
    /// </summary>
    public bool UseItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"UseItem Failed: {error}");
            return false;
        }

        // !useItem <item> [quantity] [target]
        // Strategy:
        // input1 could be quantity (int) OR target (string) if quantity is omitted (default 1)
        // Check input1 type.

        if (!GetInputString(0, "item_name", true, out string itemName, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !useItem <item_name> [quantity] [target_user]");
            return true;
        }

        int quantity = 1;
        string targetUsername = "";
        
        // Check input1
        if (CPH.TryGetArg("input1", out string input1) && !string.IsNullOrWhiteSpace(input1))
        {
            if (int.TryParse(input1, out int q))
            {
                quantity = q;
                // If input1 is quantity, input2 might be target
                GetInputString(2, "target_user", false, out targetUsername, ref error);
            }
            else
            {
                // input1 is NOT a number, so it must be target. Quantity is default 1.
                targetUsername = input1;
            }
        }

        try
        {
            var result = client.UseItem(platform, platformId, username, itemName, quantity, targetUsername).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"UseItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
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
    /// Command: !upgradeItem <item_name> [quantity]
    /// </summary>
    public bool UpgradeItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"UpgradeItem Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !upgradeItem <item_name> [quantity]");
             return true;
        }

        try
        {
            var result = client.UpgradeItem(platform, platformId, username, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"UpgradeItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Disassemble an item to get materials
    /// Command: !disassembleItem <item_name> [quantity]
    /// </summary>
    public bool DisassembleItem()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
             CPH.LogWarn($"DisassembleItem Failed: {error}");
             return false;
        }

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !disassembleItem <item_name> [quantity]");
             return true;
        }

        try
        {
            var result = client.DisassembleItem(platform, platformId, username, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"DisassembleItem API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
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
    /// Command: !startGamble <lootbox_name> [quantity]
    /// </summary>
    public bool StartGamble()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
             CPH.LogWarn($"StartGamble Failed: {error}");
             return false;
        }

        if (!GetInputString(0, "lootbox_name", true, out string lootboxItemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !startGamble <lootbox_name> [quantity]");
             return true;
        }

        try
        {
            var result = client.StartGamble(platform, platformId, username, lootboxItemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"StartGamble API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Join an existing gamble session
    /// Command: !joinGamble <gamble_id> <lootbox_name> [quantity]
    /// </summary>
    public bool JoinGamble()
    {
        EnsureInitialized();
        string error = null;
        
        // Use ValidateContext for platform details
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
             CPH.LogWarn($"JoinGamble Failed: {error}");
             return false;
        }
        
        // Note: gamble_id comes from input0
        if (!GetInputString(0, "gamble_id", true, out string gambleId, ref error) ||
            !GetInputString(1, "lootbox_name", true, out string lootboxItemName, ref error) ||
            !GetInputInt(2, "quantity", 1, out int quantity, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !joinGamble <gamble_id> <lootbox_name> [quantity]");
             return true;
        }

        try
        {
            var result = client.JoinGamble(gambleId, platform, platformId, username, lootboxItemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"JoinGamble API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
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

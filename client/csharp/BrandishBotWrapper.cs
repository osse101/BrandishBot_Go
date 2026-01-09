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
    
    // Compiled regex for username validation (performance optimization)
    private static readonly System.Text.RegularExpressions.Regex UsernameRegex = 
        new System.Text.RegularExpressions.Regex(@"^[a-zA-Z0-9_]+$", System.Text.RegularExpressions.RegexOptions.Compiled);

    // Initialize the client (called automatically on first use)
    private void EnsureInitialized()
    {
        try
        {
            // Check if client is null or from unloaded AppDomain (hot reload)
            if (client == null)
            {
                string baseUrl = "http://127.0.0.1:8080";
                //string baseUrl = CPH.GetGlobalVar<string>("ServerBaseURL", persisted:true);
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
        catch (AppDomainUnloadedException)
        {
            // Hot reload occurred - reset client and retry
            CPH.LogInfo("Detected hot reload, reinitializing client...");
            client = null;
            EnsureInitialized(); // Retry
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
            // For usernames, validate alphanumeric + underscore (filters invisible chars like U+034F)
            string trimmed = inputVal.Trim();
            if (!string.IsNullOrEmpty(trimmed) && UsernameRegex.IsMatch(trimmed))
            {
                value = trimmed;
                return true;
            }
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
    /// Helper: Format inventory JSON response for readability
    /// </summary>
    private string FormatInventory(string jsonResponse)
    {
        try
        {
            var inventory = Newtonsoft.Json.Linq.JObject.Parse(jsonResponse);
            var items = inventory["items"];
            var formattedItems = new System.Collections.Generic.List<string>();

            // Parse all items
            foreach (var item in items)
            {
                string name = item["name"].ToString();
                int qty = (int)item["quantity"];

                // Money gets special treatment - always first, emoji only
                if (name == "money")
                {
                    formattedItems.Insert(0, $"💰 {qty}");
                }
                else
                {
                    formattedItems.Add($"{qty}x {name}");
                }
            }

            return formattedItems.Count > 0 
                ? string.Join(" | ", formattedItems)
                : "Empty inventory";
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"FormatInventory failed: {ex.Message}");
            return jsonResponse; // Return raw JSON if parsing fails
        }
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
    /// Note: auto-called on first interaction
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
    /// Uses: userType, userId, userName (from streamer.bot context)
    /// Note: Will use username-based lookup if provided
    /// </summary>
    public bool GetInventory()
    {
        EnsureInitialized();
        string error = null;
        if (!CPH.TryGetArg("userType", out string platform)) return false;

        if (GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser))
        {   //target other user
            try
            {
                var result = client.GetInventoryByUsername(platform, targetUser).Result;
                CPH.SetArgument("response", FormatInventory(result));
                return true;
            }catch(Exception ex){
                CPH.LogError($"GetInventoryByUsername failed: {ex.Message}");
                return false;
            }
        }else
        {   //target self
            try
            {
                if (!CPH.TryGetArg("userName", out string userName)) return false;
                if (!CPH.TryGetArg("userId", out string platformId)) return false;
                var result = client.GetInventory(platform, platformId, userName).Result;
                CPH.SetArgument("response", FormatInventory(result));
                return true;
            }
            catch (Exception ex)
            {
                CPH.LogError($"GetInventory failed: {ex.Message}");
                return false;
            }
        }
    }

    /// <summary>
    /// Add item to user's inventory (Admin/Streamer only)
    /// Command: !addItem <target_user> <item_name> [quantity]
    /// </summary>
    public bool AddItem()
    {
        EnsureInitialized();
        string error = null;
        
        // Get platform from context (for the platform name)
        if (!CPH.TryGetArg("userType", out string platform))
        {
            CPH.LogWarn("AddItem Failed: Missing userType");
            return false;
        }

        if (!GetInputString(0, "target_user", true, out string targetUser, ref error) ||
            !GetInputString(1, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(2, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !addItem <target_user> <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.AddItemByUsername(platform, targetUser, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (AggregateException aex)
        {
             var inner = aex.InnerException ?? aex;
             CPH.LogWarn($"AddItem Error: {inner.Message}");
             CPH.SetArgument("response", $"Error: {inner.Message}");
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
    /// Command: !removeItem <target_user> <item_name> [quantity]
    /// </summary>
    public bool RemoveItem()
    {
        EnsureInitialized();
        string error = null;
        
        // Get platform from context
        if (!CPH.TryGetArg("userType", out string platform))
        {
            CPH.LogWarn("RemoveItem Failed: Missing userType");
            return false;
        }

        if (!GetInputString(0, "target_user", true, out string targetUser, ref error) ||
            !GetInputString(1, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(2, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !removeItem <target_user> <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.RemoveItemByUsername(platform, targetUser, itemName, quantity).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (AggregateException aex)
        {
             var inner = aex.InnerException ?? aex;
             CPH.LogWarn($"RemoveItem Error: {inner.Message}");
             CPH.SetArgument("response", $"Error: {inner.Message}");
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

        try
        {
            var result = client.GiveItem(fromPlatform, fromUsername, toPlatform, toUsername, itemName, quantity).Result;
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

    #region Stats & Leaderboards

    /// <summary>
    /// Get user statistics (Self)
    /// Command: !stats
    /// </summary>
    public bool GetUserStats()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"GetUserStats Failed: {error}");
            return false;
        }

        try
        {
            var result = client.GetUserStats(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUserStats API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get system-wide statistics
    /// Command: !serverStats
    /// </summary>
    public bool GetSystemStats()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetSystemStats().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetSystemStats API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get leaderboard
    /// Command: !leaderboard [metric] [limit]
    /// </summary>
    public bool GetLeaderboard()
    {
        EnsureInitialized();
        string error = null;
        
        // Defaults
        string metric = "engagement_score";
        int limit = 10;

        if (GetInputString(0, "metric", false, out string inputMetric, ref error))
        {
            metric = inputMetric;
        }
        
        if (GetInputInt(1, "limit", 10, out int inputLimit, ref error))
        {
            limit = inputLimit;
        }

        try
        {
            var result = client.GetLeaderboard(metric, limit).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetLeaderboard API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Check timeout status for a user
    /// Command: !checkTimeout [username]
    /// </summary>
    public bool GetUserTimeout()
    {
        EnsureInitialized();
        string error = null;
        
        string targetUser = null;
        // Try getting from input0, else fallback to context username
        if (!GetInputString(0, "username", false, out targetUser, ref error))
        {
            // If internal error, though false implies 'not found' here mostly?
            // GetInputString returns true if found OR not required. 
            // It returns false only if required and missing.
            // So targetUser is null here if missing.
        }

        if (string.IsNullOrEmpty(targetUser))
        {
            // Fallback to self
            CPH.TryGetArg("userName", out targetUser);
        }

        if (string.IsNullOrEmpty(targetUser))
        {
            CPH.SetArgument("response", "Usage: !checkTimeout [username] (or use in chat for self)");
            return true;
        }

        try
        {
            var result = client.GetUserTimeout(targetUser).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUserTimeout API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Progression System

    /// <summary>
    /// Get progression tree structure
    /// Command: !tree
    /// </summary>
    public bool GetProgressionTree()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetProgressionTree().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetProgressionTree API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get available (unlockable) progression nodes
    /// Command: !nodes
    /// </summary>
    public bool GetAvailableNodes()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetAvailableNodes().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetAvailableNodes API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Vote to unlock a progression node
    /// Command: !vote <node_key>
    /// </summary>
    public bool VoteForNode()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"VoteForNode Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "node_key", true, out string nodeKey, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !vote <node_key>");
            return true;
        }

        try
        {
            var result = client.VoteForNode(platform, platformId, nodeKey).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"VoteForNode API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get progression status
    /// Command: !progression
    /// </summary>
    public bool GetProgressionStatus()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetProgressionStatus().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetProgressionStatus API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get user engagement breakdown
    /// Command: !engagement [user_id] (defaults to self)
    /// </summary>
    public bool GetUserEngagement()
    {
        EnsureInitialized();
        string error = null;

        string targetId = null;
        // NOTE: Input might be a username if coming from chat, but API expects userId (platform_id)
        // If we want to support !engagement @User, we technically need the ID.
        // For now, let's assume if arg is provided, it's an ID (advanced usage) or we default to self.
        
        if (!GetInputString(0, "user_id", false, out targetId, ref error))
        {
             // no arg, use self
        }

        if (string.IsNullOrEmpty(targetId))
        {
             if (!CPH.TryGetArg("userId", out targetId))
             {
                 CPH.SetArgument("response", "Error: No user ID found in context or arguments.");
                 return true;
             }
        }

        try
        {
            var result = client.GetUserEngagement(targetId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUserEngagement API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get contribution leaderboard
    /// Command: !topContributors
    /// </summary>
    public bool GetContributionLeaderboard()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetContributionLeaderboard().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetContributionLeaderboard API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get current voting session details
    /// Command: !votingSession
    /// </summary>
    public bool GetVotingSession()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetVotingSession().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetVotingSession API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get unlock progress for the current voting session
    /// Command: !unlockProgress
    /// </summary>
    public bool GetUnlockProgress()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetUnlockProgress().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUnlockProgress API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Progression Admin

    /// <summary>
    /// Admin: Unlock a specific node
    /// Command: !adminUnlock <node_key> [level]
    /// </summary>
    public bool AdminUnlockNode()
    {
        EnsureInitialized();
        string error = null;

        if (!GetInputString(0, "node_key", true, out string nodeKey, ref error) ||
            !GetInputInt(1, "level", 1, out int level, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !adminUnlock <node_key> [level]");
             return true;
        }

        try
        {
            var result = client.AdminUnlockNode(nodeKey, level).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminUnlockNode API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: Re-lock a specific node
    /// Command: !adminRelock <node_key> [level]
    /// </summary>
    public bool AdminRelockNode()
    {
        EnsureInitialized();
        string error = null;

        if (!GetInputString(0, "node_key", true, out string nodeKey, ref error) ||
            !GetInputInt(1, "level", 1, out int level, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !adminRelock <node_key> [level]");
             return true;
        }

        try
        {
            var result = client.AdminRelockNode(nodeKey, level).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminRelockNode API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: Instantly unlock the current vote leader
    /// Command: !adminInstantUnlock
    /// </summary>
    public bool AdminInstantUnlock()
    {
        EnsureInitialized();
        try
        {
            var result = client.AdminInstantUnlock().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminInstantUnlock API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: Start a new voting session
    /// Command: !adminStartVoting
    /// </summary>
    public bool AdminStartVoting()
    {
        EnsureInitialized();
        try
        {
            var result = client.AdminStartVoting().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminStartVoting API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: End the current voting session
    /// Command: !adminEndVoting
    /// </summary>
    public bool AdminEndVoting()
    {
        EnsureInitialized();
        try
        {
            var result = client.AdminEndVoting().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminEndVoting API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: Reset the entire progression system
    /// Command: !adminReset <reason> [preserve_users]
    /// </summary>
    public bool AdminResetProgression()
    {
        EnsureInitialized();
        string error = null;

        if (!GetInputString(0, "reason", true, out string reason, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !adminReset <reason> [preserve_users(true/false)]");
             return true;
        }

        bool preserve = true;
        if (GetInputString(1, "preserve_users", false, out string preserveStr, ref error))
        {
            bool.TryParse(preserveStr, out preserve);
        }
        
        // Context for who reset it
        CPH.TryGetArg("userName", out string resetBy);
        if (string.IsNullOrEmpty(resetBy)) resetBy = "StreamerBot";

        try
        {
            var result = client.AdminResetProgression(resetBy, reason, preserve).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminResetProgression API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Admin: Add contribution points manually
    /// Command: !adminAddContribution <amount>
    /// </summary>
    public bool AdminAddContribution()
    {
        EnsureInitialized();
        string error = null;

        if (!GetInputInt(0, "amount", 0, out int amount, ref error))
        {
             // 0 is default from helper but amount is required here really, or 0 adds 0 which is safe.
             // If GetInputInt returned false that means it was malformed if it existed.
             // If it didn't exist, it returned true with default 0.
        }
        
        if (amount == 0 && CPH.TryGetArg("input0", out string _))
        {
             // If they typed something and we got 0 (and no error), it means default?
             // Helper returns false if malformed.
             // If they typed nothing, amount is 0.
             // We probably want to enforce non-zero.
             if (!CPH.TryGetArg("input0", out string s) || string.IsNullOrWhiteSpace(s))
             {
                 CPH.SetArgument("response", "Usage: !adminAddContribution <amount>");
                 return true;
             }
        }

        try
        {
            var result = client.AdminAddContribution(amount).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminAddContribution API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Jobs System

    /// <summary>
    /// Get all available jobs
    /// Command: !jobs
    /// </summary>
    public bool GetAllJobs()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetAllJobs().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetAllJobs API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get user's job progress
    /// Command: !myJobs
    /// </summary>
    public bool GetUserJobs()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"GetUserJobs Failed: {error}");
            return false;
        }

        try
        {
            var result = client.GetUserJobs(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUserJobs API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Award XP to a user (Streamer/Admin only)
    /// Command: !awardXp <username> <job_name> <amount>
    /// </summary>
    public bool AwardJobXP()
    {
        EnsureInitialized();
        string error = null;
        
        // Context is admin
        if (!ValidateContext(out string platform, out string platformId, out string _, ref error))
        {
            CPH.LogWarn($"AwardJobXP Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "username", true, out string targetUser, ref error) ||
            !GetInputString(1, "job_name", true, out string jobName, ref error) ||
            !GetInputInt(2, "amount", 0, out int amount, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !awardXp <username> <job_name> <amount>");
            return true;
        }

        try
        {
            var result = client.AwardJobXP(platform, platformId, targetUser, jobName, amount).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AwardJobXP API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get the active bonus for a job (e.g., search_bonus)
    /// Command: !jobBonus <job_key> <bonus_type>
    /// </summary>
    public bool GetJobBonus()
    {
        EnsureInitialized();
        string error = null;
        
        if (!ValidateContext(out string _, out string platformId, out string _, ref error))
        {
             CPH.LogWarn($"GetJobBonus Failed: {error}");
             return false;
        }

        if (!GetInputString(0, "job_key", true, out string jobKey, ref error) ||
            !GetInputString(1, "bonus_type", true, out string bonusType, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !jobBonus <job_key> <bonus_type>");
            return true;
        }

        try
        {
            var result = client.GetJobBonus(platformId, jobKey, bonusType).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetJobBonus API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Award XP to a user for a specific job (Admin only)
    /// Command: !awardXP <target_user> <job_key> <amount>
    /// </summary>
    public bool AdminAwardXP()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"AdminAwardXP Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "target_user", true, out string targetUser, ref error) ||
            !GetInputString(1, "job_key", true, out string jobKey, ref error) ||
            !GetInputInt(2, "amount", 1, out int amount, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !awardXP <target_user> <job_key> <amount>");
            return true;
        }

        try
        {
            var result = client.AdminAwardXP(platform, targetUser, jobKey, amount).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"AdminAwardXP API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get unlocked crafting recipes for the calling user
    /// Uses: userType, userId, userName (from streamer.bot context)
    /// </summary>
    public bool GetUnlockedRecipes()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"GetUnlockedRecipes Failed: {error}");
            return false;
        }

        try
        {
            var result = client.GetUnlockedRecipes(platform, platformId, username).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetUnlockedRecipes API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Account Linking

    /// <summary>
    /// Initiate account linking process
    /// Command: !linkAccount
    /// </summary>
    public bool InitiateLinking()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"InitiateLinking Failed: {error}");
            return false;
        }

        try
        {
            var result = client.InitiateLinking(platform, platformId, username).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"InitiateLinking API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Claim a linking code from another platform
    /// Command: !claimCode <code>
    /// </summary>
    public bool ClaimLinkingCode()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"ClaimLinkingCode Failed: {error}");
            return false;
        }

        if (!GetInputString(0, "code", true, out string code, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !claimCode <code>");
            return true;
        }

        try
        {
            var result = client.ClaimLinkingCode(platform, platformId, username, code).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"ClaimLinkingCode API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Confirm account linking
    /// Command: !confirmLink
    /// </summary>
    public bool ConfirmLinking()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string _, ref error))
        {
            CPH.LogWarn($"ConfirmLinking Failed: {error}");
            return false;
        }

        try
        {
            var result = client.ConfirmLinking(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"ConfirmLinking API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Unlink accounts
    /// Command: !unlink
    /// </summary>
    public bool UnlinkAccounts()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string _, ref error))
        {
            CPH.LogWarn($"UnlinkAccounts Failed: {error}");
            return false;
        }

        try
        {
            var result = client.UnlinkAccounts(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"UnlinkAccounts API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Get linking status for a user
    /// Command: !linkStatus
    /// </summary>
    public bool GetLinkingStatus()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string _, ref error))
        {
            CPH.LogWarn($"GetLinkingStatus Failed: {error}");
            return false;
        }

        try
        {
            var result = client.GetLinkingStatus(platform, platformId).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"GetLinkingStatus API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    #endregion

    #region Admin Utilities

    /// <summary>
    /// Reload item name aliases
    /// Command: !reloadAliases
    /// </summary>
    public bool ReloadAliases()
    {
        EnsureInitialized();
        try
        {
            var result = client.ReloadAliases().Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"ReloadAliases API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
        }
    }

    /// <summary>
    /// Test endpoint
    /// Command: !test
    /// </summary>
    public bool Test()
    {
        EnsureInitialized();
        string error = null;

        if (!ValidateContext(out string platform, out string platformId, out string username, ref error))
        {
            CPH.LogWarn($"Test Failed: {error}");
            return false;
        }

        try
        {
            var result = client.Test(platform, platformId, username).Result;
            CPH.SetArgument("response", result);
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogWarn($"Test API Error: {ex.Message}");
            CPH.SetArgument("response", $"Error: {ex.Message}");
            return true;
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

    #region Health Check

    /// <summary>
    /// Health check endpoint
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

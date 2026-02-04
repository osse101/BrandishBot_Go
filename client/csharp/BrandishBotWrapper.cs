using System;
using System.Collections.Generic;
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

    private static bool lastDevEnabled = false;
    private static string lastDevUrl = "";

    // Initialize the client (called automatically on first use)
    private void EnsureInitialized()
    {
        try
        {
            if (client == null)
            {
                string baseUrl = CPH.GetGlobalVar<string>("ServerBaseURL", persisted:true);
                string apiKey = CPH.GetGlobalVar<string>("ServerApiKey", persisted:true);
                if (string.IsNullOrEmpty(baseUrl)) baseUrl = "http://127.0.0.1:8080";
                
                if (string.IsNullOrEmpty(apiKey))
                {
                    CPH.LogError("CONFIGURATION ERROR: ServerApiKey global variable is not set!");
                    throw new InvalidOperationException("ServerApiKey not configured");
                }
                
                BrandishBotClient.Initialize(baseUrl, apiKey);
                client = BrandishBotClient.Instance;
            }

            // Sync Dev Client State (Allows turning it off/on dynamically)
            bool devEnabled = CPH.GetGlobalVar<bool>("BrandishBot_DevEnabled", persisted:true);
            string devUrl = CPH.GetGlobalVar<string>("DevBaseURL", persisted:true);

            if (devEnabled != lastDevEnabled || devUrl != lastDevUrl)
            {
                if (devEnabled)
                {
                    string apiKey = CPH.GetGlobalVar<string>("ServerApiKey", persisted: true);
                    if (!string.IsNullOrEmpty(devUrl) && !string.IsNullOrEmpty(apiKey))
                    {
                        var devClient = new BrandishBotClient(devUrl, apiKey, isForwardingInstance: true);
                        client.SetForwardingClient(devClient);
                        CPH.LogInfo($"[BrandishBot] Dev Forwarding ENABLED -> {devUrl}");
                    }
                }
                else
                {
                    client.SetForwardingClient(null);
                    CPH.LogInfo("[BrandishBot] Dev Forwarding DISABLED");
                }
                lastDevEnabled = devEnabled;
                lastDevUrl = devUrl;
            }
        }
        catch (AppDomainUnloadedException)
        {
            client = null;
            EnsureInitialized();
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
    /// Helper: Check if exception is a 403 Forbidden error
    /// </summary>
    private bool IsForbiddenError(Exception ex)
    {
        if (ex == null) return false;
        string message = GetErrorMessage(ex);
        return message.Contains("403") || message.Contains("Forbidden");
    }

    /// <summary>
    /// Helper: Check if exception is a 429 Too Many Requests (cooldown) error
    /// </summary>
    private bool IsTooManyRequestsError(Exception ex)
    {
        if (ex == null) return false;
        string message = GetErrorMessage(ex);
        return message.Contains("429") || message.Contains("Too Many Requests");
    }

    /// <summary>
    /// Helper: Get the most meaningful error message from an exception
    /// Unwraps AggregateException to get the actual inner error
    /// </summary>
    private string GetErrorMessage(Exception ex)
    {
        if (ex == null) return "Unknown error";
        
        // Unwrapping AggregateException which occurs when using .Result on async tasks
        if (ex is AggregateException aex && aex.InnerException != null)
        {
            return aex.InnerException.Message;
        }
        
        return ex.Message;
    }

    /// <summary>
    /// Helper: Strip HTTP status code prefix from error message
    /// Format: "403 Forbidden: Actual Message" -> "Actual Message"
    /// </summary>
    private string StripStatusCode(string message)
    {
        if (string.IsNullOrEmpty(message)) return message;
        int colonIndex = message.IndexOf(": ");
        // If it starts with a number (HTTP Status Code) and has a colon, strip the prefix
        if (colonIndex > 0 && char.IsDigit(message[0]))
        {
            return message.Substring(colonIndex + 2);
        }
        return message;
    }

    private void LogException(string context, Exception ex)
    {
        string message = GetErrorMessage(ex);
        CPH.LogError($"{context} failed: {message}");
    }

    private void LogWarning(string context, Exception ex)
    {
        string message = GetErrorMessage(ex);
        CPH.LogWarn($"{context} Error: {message}");
    }

    #region Version    /// <summary>
    /// Get the backend version
    /// Args: (none)
    /// </summary>
    public bool GetVersion()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetVersion().Result;
            var formatted = ResponseFormatter.FormatVersion(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogException("GetVersion", ex);
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
            client.RegisterUser(platform, platformId, username).Wait();
            CPH.SetArgument("response", "Registration successful!");
            return true;
        }
        catch (Exception ex)
        {
            LogException("RegisterUser", ex);
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

        bool isTargetMode = GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser);
        
        try
        {
            GetInventoryResponse result;
            if (isTargetMode)
            {
                result = client.GetInventoryByUsername(platform, targetUser).Result;
            }
            else
            {
                if (!CPH.TryGetArg("userName", out string userName)) return false;
                if (!CPH.TryGetArg("userId", out string platformId)) return false;
                result = client.GetInventory(platform, platformId, userName).Result;
            }

            CPH.SetArgument("response", ResponseFormatter.FormatInventory(result));
            return true;
        }
        catch (Exception ex)
        {
            string message = GetErrorMessage(ex);
            LogWarning(isTargetMode ? "GetInventoryByUsername" : "GetInventory", ex);
            
            if (message.Contains("not found") || message.Contains("404"))
            {
                CPH.SetArgument("response", isTargetMode ? $"User not found: {targetUser}" : "User not registered.");
            }
            else
            {
                CPH.SetArgument("response", $"Error: {StripStatusCode(message)}");
            }
            return true;
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !addItem <target_user> <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.AddItemByUsername(platform, targetUser, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AddItem", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !removeItem <target_user> <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.RemoveItemByUsername(platform, targetUser, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("RemoveItem", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !giveItem <target_user> <item_name> [quantity]");
            return true;
        }

        string toPlatform = fromPlatform; 

        try
        {
            var result = client.GiveItem(fromPlatform, fromPlatformId, fromUsername, toPlatform, toUsername, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GiveItem", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error)) return false;

        if (!GetInputString(0, "item_name", true, out string itemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !buyItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.BuyItem(platform, platformId, username, itemName, quantity).Result;
            CPH.SetArgument("response", ResponseFormatter.FormatMessage(result));
            return true;
        }
        catch (Exception ex)
        {
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !sellItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.SellItem(platform, platformId, username, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("SellItem", ex);
                CPH.SetArgument("response", errorMsg);
            }
            return true;
        }
    }

    /// <summary>
    /// Get current item sell prices
    /// Args: (none)
    /// </summary>
    public bool GetSellPrices()
    {
        EnsureInitialized();

        try
        {
            var result = client.GetSellPrices().Result;
            var formatted = ResponseFormatter.FormatPrices(result, "Sell");
            CPH.SetArgument("response", formatted);
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
            var formatted = ResponseFormatter.FormatPrices(result, "Buy");
            CPH.SetArgument("response", formatted);
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !useItem <item_name> [quantity] [target_user]");
            return true;
        }

        try
        {
            var result = client.UseItem(platform, platformId, username, itemName, quantity, targetUsername).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("UseItem", ex);
                CPH.SetArgument("response", $"Error: {errorMsg}");
            }
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
                return true;
            }
            LogException("Search", ex);
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !upgradeItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.UpgradeItem(platform, platformId, username, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("UpgradeItem", ex);
                CPH.SetArgument("response", errorMsg);
            }
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
        if( quantity < 1 )
        {
            CPH.SetArgument("response", "Invalid quantity. Usage: !disassembleItem <item_name> [quantity]");
            return true;
        }

        try
        {
            var result = client.DisassembleItem(platform, platformId, username, itemName, quantity).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("DisassembleItem", ex);
                CPH.SetArgument("response", errorMsg);
            }
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
            CPH.SetArgument("response", ResponseFormatter.FormatRecipes(result));
            return true;
        }
        catch (Exception ex)
        {
            LogException("GetRecipes", ex);
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
        if (!ValidateContext(out string platform, out string platformId, out string username, ref error)) return false;

        if (!GetInputString(0, "lootbox_name", true, out string lootboxItemName, ref error) ||
            !GetInputInt(1, "quantity", 1, out int quantity, ref error))
        {
             CPH.SetArgument("response", $"{error} Usage: !startGamble <lootbox_name> [quantity]");
             return true;
        }

        try
        {
            var result = client.StartGamble(platform, platformId, username, lootboxItemName, quantity).Result;
            if (!string.IsNullOrEmpty(result.GambleId))
            {
                CPH.SetGlobalVar("gambleId", result.GambleId, persisted: false);
            }
            CPH.SetArgument("response", ResponseFormatter.FormatMessage(result));
            return true;
        }
        catch (Exception ex)
        {
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Join an existing gamble session
    /// Command: !joinGamble <gamble_id>
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

        string gambleId = CPH.GetGlobalVar<string>("gambleId", persisted:false);
        
        try
        {
            var result = client.JoinGamble(gambleId, platform, platformId, username).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("JoinGamble", ex);
                CPH.SetArgument("response", errorMsg);
            }
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
            LogException("GetActiveGamble", ex);
            return false;
        }
    }

    #endregion

    #region Stats & Leaderboards

    /// <summary>
    /// Get user statistics
    /// Command: !stats [target_user]
    /// </summary>
    public bool GetUserStats()
    {
        EnsureInitialized();
        string error = null;

        if (!CPH.TryGetArg("userType", out string platform))
        {
            CPH.LogWarn("GetUserStats Failed: Missing userType");
            return false;
        }

        // Check for target_user parameter (optional)
        if (GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser))
        {
            // Target-mode: query another user by username
            try
            {
                var result = client.GetUserStatsByUsername(platform, targetUser).Result;
                CPH.SetArgument("response", result);
                return true;
            }
            catch (Exception ex)
            {
                LogWarning("GetUserStatsByUsername", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
        }
        else
        {
            // Self-mode: query own stats
            if (!ValidateContext(out string _, out string platformId, out string username, ref error))
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
                LogWarning("GetUserStats", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
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
            LogWarning("GetSystemStats", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
        
        string metric = "engagement_score";
        if (GetInputString(0, "metric", false, out string inputMetric, ref error)) metric = inputMetric;
        
        if (!GetInputInt(1, "limit", 10, out int limit, ref error)) limit = 10;

        try
        {
            var result = client.GetLeaderboard(metric, limit: limit).Result;
            CPH.SetArgument("response", ResponseFormatter.FormatLeaderboard(result, metric));
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetLeaderboard", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Check timeout status for a user (gett command)
    /// Command: !gett [username]
    /// </summary>
    public bool GetUserTimeout()
    {
        EnsureInitialized();
        string error = null;
        string targetUser = null;
        
        GetInputString(0, "username", false, out targetUser, ref error);
        if (string.IsNullOrEmpty(targetUser))
        {
            // Fallback to self
            CPH.TryGetArg("userName", out targetUser);
        }

        if (!CPH.TryGetArg("userType", out string platform)) platform = "twitch";

        try
        {
            var result = client.GetUserTimeout(platform, targetUser).Result;
            var jsonResult = Newtonsoft.Json.Linq.JObject.Parse(result);
            bool isTimedOut = jsonResult.Value<bool>("is_timed_out");
            double remainingSeconds = jsonResult.Value<double>("remaining_seconds");

            if (isTimedOut)
            {
                int minutes = (int)(remainingSeconds / 60);
                int seconds = (int)(remainingSeconds % 60);
                CPH.SetArgument("response", $"{targetUser} is timed out for {minutes}m {seconds}s");
            }
            else
            {
                CPH.SetArgument("response", $"{targetUser} is not timed out");
            }
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetUserTimeout", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            LogWarning("GetProgressionTree", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            LogWarning("GetAvailableNodes", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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

        if (!GetInputInt(0, "node_key", 0, out int nodeKey, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !vote <node_key>");
            return true;
        }

        List<string> activeNodes = CPH.GetGlobalVar<List<string>>("ActiveNodes");
        if(nodeKey <= 0 || nodeKey > activeNodes.Count)
        {
            CPH.SetArgument("response", "Invalid node key. Must be between 1 and " + activeNodes.Count);
            return true;
        }
        string nodeKeyString = activeNodes[nodeKey-1];

        try
        {
            var result = client.VoteForNode(platform, platformId, nodeKeyString).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("VoteForNode", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            CPH.SetArgument("response", ResponseFormatter.FormatProgressionStatus(result));
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetProgressionStatus", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Get user engagement breakdown
    /// Command: !engagement [target_user] (defaults to self)
    /// Note: Will use username-based lookup if provided
    /// </summary>
    public bool GetUserEngagement()
    {
        EnsureInitialized();
        string error = null;
        if (!CPH.TryGetArg("userType", out string platform)) return false;

        if (GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser))
        {   //target other user
            try
            {
                var result = client.GetUserEngagementByUsername(platform, targetUser).Result;
                CPH.SetArgument("response", result);
                return true;
            }catch(Exception ex){
                CPH.LogError($"GetUserEngagementByUsername failed: {ex.Message}");
                return false;
            }
        }else
        {   //target self
            try
            {
                if (!CPH.TryGetArg("userId", out string platformId)) return false;
                var result = client.GetUserEngagement(platform, platformId).Result;
                CPH.SetArgument("response", result);
                return true;
            }
            catch (Exception ex)
            {
                CPH.LogError($"GetUserEngagement failed: {ex.Message}");
                return false;
            }
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
            CPH.SetArgument("response", ResponseFormatter.FormatVotingOptions(result));
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetVotingSession", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Get unlock progress for the current voting session
    /// Command: !unlockProgress or !treeprogress
    /// </summary>
    public bool GetUnlockProgress()
    {
        EnsureInitialized();
        try
        {
            var result = client.GetUnlockProgress().Result;
            CPH.SetArgument("response", ResponseFormatter.FormatUnlockProgress(result));
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetUnlockProgress", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminUnlockNode", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Admin: Unlock ALL progression nodes at max level (DEBUG ONLY)
    /// Command: !adminUnlockAll
    /// </summary>
    public bool AdminUnlockAllNodes()
    {
        EnsureInitialized();

        try
        {
            var result = client.AdminUnlockAllNodes().Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminUnlockAllNodes", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminRelockNode", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminStartVoting", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminEndVoting", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminResetProgression", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("AdminAddContribution", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    #endregion

    #region Jobs System

    /// <summary>
    /// Get user's job progress
    /// Command: !myJobs [target_user]
    /// </summary>
    public bool GetUserJobs()
    {
        EnsureInitialized();
        string error = null;

        if (!CPH.TryGetArg("userType", out string platform))
        {
            CPH.LogWarn("GetUserJobs Failed: Missing userType");
            return false;
        }

        // Check for target_user parameter (optional)
        if (GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser))
        {
            // Target-mode: query another user by username
            try
            {
                var result = client.GetUserJobsByUsername(platform, targetUser).Result;
                CPH.SetArgument("response", result);
                return true;
            }
            catch (Exception ex)
            {
                LogWarning("GetUserJobsByUsername", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
        }
        else
        {
            // Self-mode: query own jobs
            if (!ValidateContext(out string _, out string platformId, out string username, ref error))
            {
                CPH.LogWarn($"GetUserJobs Failed: {error}");
                return false;
            }

            try
            {
                var result = client.GetUserJobs(platform, platformId, username).Result;
                CPH.SetArgument("response", result);
                return true;
            }
            catch (Exception ex)
            {
                LogWarning("GetUserJobs", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
        }
    }

    /// <summary>
    /// Award XP to a user (Streamer/Admin only)
    /// Command: !awardXp <username> <job_key> <amount>
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
            !GetInputString(1, "job_key", true, out string jobKey, ref error) ||
            !GetInputInt(2, "amount", 0, out int amount, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !awardXp <username> <job_key> <amount>");
            return true;
        }

        try
        {
            var result = client.AwardJobXP(platform, targetUser, jobKey, amount).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            string errorMsg = StripStatusCode(GetErrorMessage(ex));
            if (IsForbiddenError(ex) || IsTooManyRequestsError(ex))
            {
                CPH.SetArgument("response", errorMsg);
            }
            else
            {
                LogWarning("AwardJobXP", ex);
                CPH.SetArgument("response", errorMsg);
            }
            return true;
        }
    }



    /// <summary>
    /// Get unlocked crafting recipes for the calling user
    /// Uses: userType, userId, userName (from streamer.bot context)
    /// </summary>
    /// <summary>
    /// Get unlocked recipes for a user
    /// Command: !recipes [target_user]
    /// </summary>
    public bool GetUnlockedRecipes()
    {
        EnsureInitialized();
        string error = null;

        if (!CPH.TryGetArg("userType", out string platform))
        {
            CPH.LogWarn("GetUnlockedRecipes Failed: Missing userType");
            return false;
        }

        // Check for target_user parameter (optional)
        if (GetInputString(0, "target_user", true, out string targetUser, ref error) && !string.IsNullOrWhiteSpace(targetUser))
        {
            // Target-mode: query another user by username
            try
            {
                var result = client.GetUnlockedRecipesByUsername(platform, targetUser).Result;
                CPH.SetArgument("response", ResponseFormatter.FormatRecipes(result));
                return true;
            }
            catch (Exception ex)
            {
                LogWarning("GetUnlockedRecipesByUsername", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
        }
        else
        {
            // Self-mode: query own recipes
            if (!ValidateContext(out string _, out string platformId, out string username, ref error))
            {
                CPH.LogWarn($"GetUnlockedRecipes Failed: {error}");
                return false;
            }

            try
            {
                var result = client.GetUnlockedRecipes(platform, platformId, username).Result;
                CPH.SetArgument("response", ResponseFormatter.FormatRecipes(result));
                return true;
            }
            catch (Exception ex)
            {
                LogWarning("GetUnlockedRecipes", ex);
                CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
                return true;
            }
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("InitiateLinking", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("ClaimLinkingCode", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("ConfirmLinking", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
            return true;
        }
    }

    /// <summary>
    /// Unlink accounts
    /// Command: !unlink <target_platform>
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

        if (!GetInputString(0, "target_platform", true, out string targetPlatform, ref error))
        {
            CPH.SetArgument("response", $"{error} Usage: !unlink <target_platform>");
            return true;
        }

        try
        {
            var result = client.UnlinkAccounts(platform, platformId, targetPlatform).Result;
            var formatted = ResponseFormatter.FormatMessage(result);
            CPH.SetArgument("response", formatted);
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("UnlinkAccounts", ex);
            CPH.SetArgument("response", StripStatusCode(GetErrorMessage(ex)));
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
            CPH.SetArgument("response", ResponseFormatter.FormatLinkingStatus(result));
            return true;
        }
        catch (Exception ex)
        {
            LogWarning("GetLinkingStatus", ex);
            CPH.SetArgument("response", $"Error: {StripStatusCode(GetErrorMessage(ex))}");
            return true;
        }
    }

    #endregion

    #region Admin Utilities



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
        catch (Exception ex)
        {
            LogException("HandleMessage", ex);
            return false;
        }
    }


    #endregion

    #region Predictions

    /// <summary>
    /// Process a Twitch prediction outcome
    ///
    /// Required Args (from Twitch Prediction Completed event):
    /// - eventSource: Platform (usually "twitch")
    /// - prediction.winningOutcome.id: Winner outcome ID
    /// - prediction.winningOutcome.title: Winner outcome title
    /// - prediction.winningOutcome.users: Number of users who predicted this outcome
    /// - prediction.winningOutcome.points: Total points spent on this outcome
    /// - prediction.outcome0.users, prediction.outcome0.points: First outcome stats
    /// - prediction.outcome1.users, prediction.outcome1.points: Second outcome stats
    ///
    /// Optional Args (for individual participant tracking):
    /// - prediction.topPredictor0.userName, prediction.topPredictor0.userId, prediction.topPredictor0.points
    /// - prediction.topPredictor1.userName, etc. (if available from custom integration)
    ///
    /// NOTE: Standard Twitch prediction events only provide aggregate data (total users/points per outcome).
    /// This wrapper creates synthetic participant entries from the aggregates.
    /// For full individual tracking, integrate with Twitch API to fetch top predictors.
    ///
    /// Returns: Sets "predictionResult" arg with contribution awarded and XP details
    /// </summary>
    public bool ProcessPredictionOutcome()
    {
        EnsureInitialized();

        try
        {
            // Get platform (default to twitch for predictions)
            if (!CPH.TryGetArg("eventSource", out string platform))
            {
                platform = "twitch";
            }

            // Get winning outcome data
            if (!CPH.TryGetArg("prediction.winningOutcome.title", out string winningTitle))
            {
                CPH.LogError("[Prediction] Missing winning outcome title");
                return false;
            }

            if (!CPH.TryGetArg("prediction.winningOutcome.points", out string winningPointsStr) ||
                !int.TryParse(winningPointsStr, out int winningPoints))
            {
                CPH.LogError("[Prediction] Missing or invalid winning outcome points");
                return false;
            }

            // Calculate total points across all outcomes
            int totalPoints = 0;
            int outcomeIndex = 0;
            var participants = new List<PredictionParticipant>();

            // Collect points from all outcomes (up to 10 outcomes, though typically 2)
            while (outcomeIndex < 10)
            {
                string pointsKey = $"prediction.outcome{outcomeIndex}.points";
                string usersKey = $"prediction.outcome{outcomeIndex}.users";
                string titleKey = $"prediction.outcome{outcomeIndex}.title";

                if (CPH.TryGetArg(pointsKey, out string pointsStr) &&
                    int.TryParse(pointsStr, out int points))
                {
                    totalPoints += points;

                    // Create synthetic participants for this outcome (aggregate entry)
                    if (CPH.TryGetArg(usersKey, out string usersStr) &&
                        int.TryParse(usersStr, out int userCount) &&
                        CPH.TryGetArg(titleKey, out string outcomeTitle))
                    {
                        // Create a synthetic participant representing this outcome's aggregate
                        participants.Add(new PredictionParticipant
                        {
                            Username = $"outcome_{outcomeIndex}_{outcomeTitle}",
                            PlatformId = "0", // Synthetic ID
                            PointsSpent = points
                        });

                        CPH.LogInfo($"[Prediction] Outcome {outcomeIndex} ({outcomeTitle}): {userCount} users, {points} points");
                    }

                    outcomeIndex++;
                }
                else
                {
                    break; // No more outcomes
                }
            }

            if (totalPoints == 0)
            {
                CPH.LogWarn("[Prediction] Total points is 0, skipping prediction processing");
                return true;
            }

            // Try to get broadcaster as the "winner" (or use first top predictor if available)
            string winnerUsername = "unknown_winner";
            string winnerPlatformId = "0";

            // Check for custom top predictor data (if integrated)
            if (CPH.TryGetArg("prediction.topPredictor0.userName", out string topPredictorName) &&
                CPH.TryGetArg("prediction.topPredictor0.userId", out string topPredictorId))
            {
                winnerUsername = topPredictorName;
                winnerPlatformId = topPredictorId;
                CPH.LogInfo($"[Prediction] Using top predictor as winner: {winnerUsername}");
            }
            else if (CPH.TryGetArg("broadcastUserName", out string broadcasterName) &&
                     CPH.TryGetArg("broadcastUserId", out string broadcasterId))
            {
                // Fallback: Use broadcaster as synthetic winner
                winnerUsername = broadcasterName;
                winnerPlatformId = broadcasterId;
                CPH.LogInfo($"[Prediction] Using broadcaster as synthetic winner: {winnerUsername}");
            }

            // Create winner object
            var winner = new PredictionWinner
            {
                Username = winnerUsername,
                PlatformId = winnerPlatformId,
                PointsWon = winningPoints
            };

            // Add individual predictors if provided (custom integration)
            int predictorIndex = 0;
            while (predictorIndex < 100) // Max 100 top predictors
            {
                string nameKey = $"prediction.topPredictor{predictorIndex}.userName";
                string idKey = $"prediction.topPredictor{predictorIndex}.userId";
                string pointsKey = $"prediction.topPredictor{predictorIndex}.points";

                if (CPH.TryGetArg(nameKey, out string predictorName) &&
                    CPH.TryGetArg(idKey, out string predictorId) &&
                    CPH.TryGetArg(pointsKey, out string predictorPointsStr) &&
                    int.TryParse(predictorPointsStr, out int predictorPoints))
                {
                    participants.Add(new PredictionParticipant
                    {
                        Username = predictorName,
                        PlatformId = predictorId,
                        PointsSpent = predictorPoints
                    });

                    predictorIndex++;
                }
                else
                {
                    break;
                }
            }

            if (predictorIndex > 0)
            {
                CPH.LogInfo($"[Prediction] Added {predictorIndex} individual predictors");
            }

            // Call the API
            CPH.LogInfo($"[Prediction] Processing outcome: {totalPoints} total points, {participants.Count} participant entries");

            var result = client.ProcessPredictionOutcome(platform, winner, totalPoints, participants).Result;

            // Set result arguments
            CPH.SetArgument("predictionResult", result.Message);
            CPH.SetArgument("contributionAwarded", result.ContributionAwarded);
            CPH.SetArgument("winnerXpAwarded", result.WinnerXpAwarded);
            CPH.SetArgument("participantsProcessed", result.ParticipantsProcessed);
            CPH.SetArgument("totalPoints", result.TotalPoints);

            CPH.LogInfo($"[Prediction] ✓ {ResponseFormatter.FormatPredictionResult(result)}");

            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"[Prediction] Failed: {ex.Message}");
            if (ex.InnerException != null)
            {
                CPH.LogError($"[Prediction] Inner: {ex.InnerException.Message}");
            }
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
            LogException("HealthCheck", ex);
            return false;
        }
    }

    #endregion
}

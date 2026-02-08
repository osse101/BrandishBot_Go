using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;
using System.Collections.Generic;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace BrandishBot.Client
{
    /// <summary>
    /// Error response from API
    /// </summary>
    public class ApiErrorResponse
    {
        [JsonProperty("error")]
        public string Error { get; set; }

        [JsonProperty("fields")]
        public Dictionary<string, string> Fields { get; set; }
    }

    /// <summary>
    /// BrandishBot API Client for streamer.bot
    /// C# 4.8 compatible HTTP client for Twitch and YouTube integrations
    /// Singleton pattern: Initialize once with Initialize(), then use Instance everywhere
    /// </summary>
    public partial class BrandishBotClient
    {
        private static BrandishBotClient _instance;
        private static readonly object _lock = new object();

        private readonly string _baseUrl;
        private readonly string _apiKey;
        private readonly HttpClient _httpClient;

        public static BrandishBotClient Instance => _instance;
        public static bool IsInitialized => _instance != null;

        public static void Initialize(string baseUrl, string apiKey, bool forceReinitialize = false)
        {
            lock (_lock)
            {
                if (forceReinitialize || _instance == null)
                {
                    _instance = new BrandishBotClient(baseUrl, apiKey);
                }
            }
        }

        private readonly bool _isForwardingInstance;
        private BrandishBotClient _forwardTo;

        public BrandishBotClient(string baseUrl, string apiKey, bool isForwardingInstance = false)
        {
            _baseUrl = baseUrl.TrimEnd('/');
            _apiKey = apiKey;
            _isForwardingInstance = isForwardingInstance;
            _httpClient = new HttpClient();
            _httpClient.DefaultRequestHeaders.Add("X-API-Key", apiKey);
        }

        public void SetForwardingClient(BrandishBotClient devClient)
        {
            _forwardTo = devClient;
        }

        private void ForwardRequest(string method, string endpoint, Func<BrandishBotClient, Task> action)
        {
            if (_isForwardingInstance || _forwardTo == null) return;
            
            Task.Run(async () =>
            {
                try { await action(_forwardTo).ConfigureAwait(false); }
                catch { /* Silent fail for dev PC */ }
            });
        }

        private async Task<T> GetAsync<T>(string endpoint)
        {
            ForwardRequest("GET", endpoint, c => c.GetAsync<T>(endpoint));
            var response = await _httpClient.GetAsync(_baseUrl + endpoint);
            return await HandleHttpResponse<T>(response);
        }

        private async Task<T> PostAsync<T>(string endpoint, object data)
        {
            ForwardRequest("POST", endpoint, c => c.PostAsync<T>(endpoint, data));
            var jsonBody = JsonConvert.SerializeObject(data);
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await _httpClient.PostAsync(_baseUrl + endpoint, content);
            return await HandleHttpResponse<T>(response);
        }

        private async Task<string> PostRawAsync(string endpoint, object data)
        {
            ForwardRequest("POST", endpoint, c => c.PostRawAsync(endpoint, data));
            var jsonBody = JsonConvert.SerializeObject(data);
            var content = new StringContent(jsonBody, Encoding.UTF8, "application/json");
            var response = await _httpClient.PostAsync(_baseUrl + endpoint, content);
            return await HandleHttpResponse(response);
        }

        private async Task<T> HandleHttpResponse<T>(HttpResponseMessage response)
        {
            string body = await response.Content.ReadAsStringAsync();
            if (response.IsSuccessStatusCode)
            {
                return JsonConvert.DeserializeObject<T>(body);
            }

            string errorMessage = ExtractErrorMessage(body, response.StatusCode);
            throw new HttpRequestException($"{(int)response.StatusCode} {response.StatusCode}: {errorMessage}");
        }

        private async Task<string> HandleHttpResponse(HttpResponseMessage response)
        {
            string body = await response.Content.ReadAsStringAsync();
            if (response.IsSuccessStatusCode) return body;

            string errorMessage = ExtractErrorMessage(body, response.StatusCode);
            throw new HttpRequestException($"{(int)response.StatusCode} {response.StatusCode}: {errorMessage}");
        }
        
        private string BuildQuery(params string[] parameters)
        {
            return "?" + string.Join("&", parameters);
        }

        private string ExtractErrorMessage(string responseBody, System.Net.HttpStatusCode statusCode)
        {
            if (string.IsNullOrWhiteSpace(responseBody))
                return GetGenericErrorMessage(statusCode);

            try
            {
                JObject json = JObject.Parse(responseBody);
                if (json["error"] != null) return json["error"].Value<string>();
                if (json["message"] != null) return json["message"].Value<string>();
            }
            catch { }

            string trimmed = responseBody.Trim();
            return (!string.IsNullOrEmpty(trimmed) && trimmed.Length < 500) ? trimmed : GetGenericErrorMessage(statusCode);
        }

        private string GetGenericErrorMessage(System.Net.HttpStatusCode statusCode)
        {
            switch (statusCode)
            {
                case System.Net.HttpStatusCode.BadRequest: return "Invalid request. Please check your inputs.";
                case System.Net.HttpStatusCode.Unauthorized: return "Authentication failed. Please check your API key.";
                case System.Net.HttpStatusCode.Forbidden: return "That feature is locked. Unlock it in the progression tree.";
                case System.Net.HttpStatusCode.NotFound: return "Resource not found.";
                case System.Net.HttpStatusCode.InternalServerError: return "Server error occurred. Please try again.";
                case System.Net.HttpStatusCode.ServiceUnavailable: return "Server is temporarily unavailable. Please try again later.";
                case (System.Net.HttpStatusCode)429: return "That action is on cooldown. Please wait a bit.";
                default: return "An error occurred. Please try again.";
            }
        }
    }

    public static class Platform
    {
        public const string Twitch = "twitch";
        public const string YouTube = "youtube";
        public const string Discord = "discord";
    }

    public static class EventType
    {
        public const string Message = "message";
        public const string Follow = "follow";
        public const string Subscribe = "subscribe";
        public const string Raid = "raid";
        public const string Bits = "bits";
        public const string Gift = "gift";
    }

    public static class ItemName
    {
        public const string Money = "money";
        public const string Junkbox = "junkbox";
        public const string Lootbox = "lootbox";
        public const string Goldbox = "goldbox";
        public const string Missile = "missile";
    }
}

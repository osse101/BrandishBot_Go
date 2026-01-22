using System;
using System.IO;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Newtonsoft.Json;

namespace BrandishBot.Client
{
    /// <summary>
    /// SSE Event received from the server
    /// </summary>
    public class SSEEvent
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("type")]
        public string Type { get; set; }

        [JsonProperty("timestamp")]
        public long Timestamp { get; set; }

        [JsonProperty("payload")]
        public object Payload { get; set; }

        /// <summary>
        /// Get payload as a specific type
        /// </summary>
        public T GetPayload<T>()
        {
            if (Payload == null) return default(T);
            var json = Payload.ToString();
            return JsonConvert.DeserializeObject<T>(json);
        }
    }

    /// <summary>
    /// Payload for job level up events
    /// </summary>
    public class JobLevelUpPayload
    {
        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("job_key")]
        public string JobKey { get; set; }

        [JsonProperty("old_level")]
        public int OldLevel { get; set; }

        [JsonProperty("new_level")]
        public int NewLevel { get; set; }

        [JsonProperty("source")]
        public string Source { get; set; }
    }

    /// <summary>
    /// Payload for voting started events
    /// </summary>
    public class VotingStartedPayload
    {
        [JsonProperty("session_id")]
        public int SessionId { get; set; }

        [JsonProperty("node_key")]
        public string NodeKey { get; set; }

        [JsonProperty("target_level")]
        public int TargetLevel { get; set; }

        [JsonProperty("auto_selected")]
        public bool AutoSelected { get; set; }

        [JsonProperty("options")]
        public VotingOptionInfo[] Options { get; set; }

        [JsonProperty("previous_unlock")]
        public string PreviousUnlock { get; set; }
    }

    /// <summary>
    /// Voting option information
    /// </summary>
    public class VotingOptionInfo
    {
        [JsonProperty("node_key")]
        public string NodeKey { get; set; }

        [JsonProperty("display_name")]
        public string DisplayName { get; set; }
    }

    /// <summary>
    /// Payload for cycle completed events
    /// </summary>
    public class CycleCompletedPayload
    {
        [JsonProperty("unlocked_node")]
        public NodeInfo UnlockedNode { get; set; }

        [JsonProperty("voting_session")]
        public VotingSessionInfo VotingSession { get; set; }
    }

    /// <summary>
    /// Node information
    /// </summary>
    public class NodeInfo
    {
        [JsonProperty("node_key")]
        public string NodeKey { get; set; }

        [JsonProperty("display_name")]
        public string DisplayName { get; set; }
    }

    /// <summary>
    /// Voting session information
    /// </summary>
    public class VotingSessionInfo
    {
        [JsonProperty("session_id")]
        public int SessionId { get; set; }

        [JsonProperty("options")]
        public VotingOptionInfo[] Options { get; set; }
    }

    /// <summary>
    /// SSE event types
    /// </summary>
    public static class SSEEventType
    {
        public const string JobLevelUp = "job.level_up";
        public const string VotingStarted = "progression.voting_started";
        public const string CycleCompleted = "progression.cycle_completed";
        public const string Keepalive = "keepalive";
        public const string Connected = "connected";
    }

    /// <summary>
    /// BrandishBot SSE Client for real-time event streaming
    /// C# 4.8 compatible async SSE client with auto-reconnect
    /// </summary>
    public class BrandishBotSSE : IDisposable
    {
        private readonly string _baseUrl;
        private readonly string _apiKey;
        private readonly string[] _eventTypes;
        private readonly HttpClient _httpClient;
        private CancellationTokenSource _cancellationTokenSource;
        private bool _disposed;

        // Event handlers
        public event EventHandler<SSEEvent> OnJobLevelUp;
        public event EventHandler<SSEEvent> OnVotingStarted;
        public event EventHandler<SSEEvent> OnCycleCompleted;
        public event EventHandler<SSEEvent> OnAnyEvent;
        public event EventHandler<Exception> OnError;
        public event EventHandler OnConnected;
        public event EventHandler OnDisconnected;

        // Connection state
        public bool IsConnected { get; private set; }
        public bool IsReconnecting { get; private set; }

        // Reconnection settings
        private const int InitialBackoffMs = 1000;
        private const int MaxBackoffMs = 30000;
        private const double BackoffMultiplier = 2.0;

        /// <summary>
        /// Creates a new BrandishBot SSE client
        /// </summary>
        /// <param name="baseUrl">Base URL of the BrandishBot API</param>
        /// <param name="apiKey">API key for authentication</param>
        /// <param name="eventTypes">Event types to subscribe to (null for all)</param>
        public BrandishBotSSE(string baseUrl, string apiKey, string[] eventTypes = null)
        {
            _baseUrl = baseUrl.TrimEnd('/');
            _apiKey = apiKey;
            _eventTypes = eventTypes;
            _httpClient = new HttpClient();
            _httpClient.DefaultRequestHeaders.Add("X-API-Key", apiKey);
            _httpClient.DefaultRequestHeaders.Add("Accept", "text/event-stream");
            _httpClient.Timeout = TimeSpan.FromMilliseconds(Timeout.Infinite);
        }

        /// <summary>
        /// Start the SSE connection with auto-reconnect
        /// </summary>
        public async Task StartAsync()
        {
            if (_cancellationTokenSource != null)
            {
                throw new InvalidOperationException("SSE client is already running");
            }

            _cancellationTokenSource = new CancellationTokenSource();
            await ConnectLoopAsync(_cancellationTokenSource.Token);
        }

        /// <summary>
        /// Start the SSE connection in the background
        /// </summary>
        public void Start()
        {
            if (_cancellationTokenSource != null)
            {
                throw new InvalidOperationException("SSE client is already running");
            }

            _cancellationTokenSource = new CancellationTokenSource();
            Task.Run(() => ConnectLoopAsync(_cancellationTokenSource.Token));
        }

        /// <summary>
        /// Stop the SSE connection
        /// </summary>
        public void Stop()
        {
            if (_cancellationTokenSource != null)
            {
                _cancellationTokenSource.Cancel();
                _cancellationTokenSource.Dispose();
                _cancellationTokenSource = null;
            }
            IsConnected = false;
            IsReconnecting = false;
        }

        private async Task ConnectLoopAsync(CancellationToken cancellationToken)
        {
            int backoffMs = InitialBackoffMs;
            int consecutiveFailures = 0;

            while (!cancellationToken.IsCancellationRequested)
            {
                try
                {
                    await ConnectAsync(cancellationToken);
                    // If we get here, connection closed normally
                    backoffMs = InitialBackoffMs;
                    consecutiveFailures = 0;
                }
                catch (OperationCanceledException)
                {
                    // Cancelled, exit loop
                    break;
                }
                catch (Exception ex)
                {
                    consecutiveFailures++;
                    IsConnected = false;
                    IsReconnecting = true;

                    OnError?.Invoke(this, ex);

                    // Wait before reconnecting with exponential backoff
                    try
                    {
                        await Task.Delay(backoffMs, cancellationToken);
                    }
                    catch (OperationCanceledException)
                    {
                        break;
                    }

                    // Increase backoff for next failure
                    backoffMs = Math.Min((int)(backoffMs * BackoffMultiplier), MaxBackoffMs);
                }
            }

            IsConnected = false;
            IsReconnecting = false;
            OnDisconnected?.Invoke(this, EventArgs.Empty);
        }

        private async Task ConnectAsync(CancellationToken cancellationToken)
        {
            string url = _baseUrl + "/api/v1/events";
            if (_eventTypes != null && _eventTypes.Length > 0)
            {
                url += "?types=" + string.Join(",", _eventTypes);
            }

            using (var response = await _httpClient.GetAsync(url, HttpCompletionOption.ResponseHeadersRead, cancellationToken))
            {
                response.EnsureSuccessStatusCode();

                IsConnected = true;
                IsReconnecting = false;
                OnConnected?.Invoke(this, EventArgs.Empty);

                using (var stream = await response.Content.ReadAsStreamAsync())
                using (var reader = new StreamReader(stream))
                {
                    await ReadEventsAsync(reader, cancellationToken);
                }
            }
        }

        private async Task ReadEventsAsync(StreamReader reader, CancellationToken cancellationToken)
        {
            string eventId = null;
            string eventType = null;
            string data = null;

            while (!reader.EndOfStream && !cancellationToken.IsCancellationRequested)
            {
                string line = await reader.ReadLineAsync();

                if (string.IsNullOrEmpty(line))
                {
                    // Empty line means end of event
                    if (!string.IsNullOrEmpty(data))
                    {
                        ProcessEvent(eventId, eventType, data);
                    }
                    eventId = null;
                    eventType = null;
                    data = null;
                    continue;
                }

                if (line.StartsWith("id: "))
                {
                    eventId = line.Substring(4);
                }
                else if (line.StartsWith("event: "))
                {
                    eventType = line.Substring(7);
                }
                else if (line.StartsWith("data: "))
                {
                    data = line.Substring(6);
                }
            }
        }

        private void ProcessEvent(string id, string eventType, string data)
        {
            if (string.IsNullOrEmpty(eventType) ||
                eventType == SSEEventType.Keepalive ||
                eventType == SSEEventType.Connected)
            {
                return; // Skip keepalive and connection events
            }

            try
            {
                SSEEvent sseEvent = JsonConvert.DeserializeObject<SSEEvent>(data);

                // Override type from event line if present
                if (!string.IsNullOrEmpty(eventType))
                {
                    sseEvent.Type = eventType;
                }
                if (!string.IsNullOrEmpty(id))
                {
                    sseEvent.Id = id;
                }

                // Fire specific handlers
                switch (eventType)
                {
                    case SSEEventType.JobLevelUp:
                        OnJobLevelUp?.Invoke(this, sseEvent);
                        break;
                    case SSEEventType.VotingStarted:
                        OnVotingStarted?.Invoke(this, sseEvent);
                        break;
                    case SSEEventType.CycleCompleted:
                        OnCycleCompleted?.Invoke(this, sseEvent);
                        break;
                }

                // Fire generic handler
                OnAnyEvent?.Invoke(this, sseEvent);
            }
            catch (Exception ex)
            {
                OnError?.Invoke(this, ex);
            }
        }

        public void Dispose()
        {
            if (!_disposed)
            {
                Stop();
                _httpClient?.Dispose();
                _disposed = true;
            }
        }
    }
}

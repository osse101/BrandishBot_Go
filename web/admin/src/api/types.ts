// Health & version
export interface HealthResponse {
  status: string;
  message?: string;
}

export interface VersionInfo {
  version: string;
  go_version: string;
  build_time?: string;
  git_commit?: string;
}

// Admin metrics
export interface AdminMetrics {
  http: {
    requests_total_by_status: Record<string, number>;
    avg_latency_ms: number;
    p95_latency_ms: number;
    in_flight: number;
  };
  events: {
    published_total_by_type: Record<string, number>;
    handler_errors_by_type: Record<string, number>;
  };
  business: {
    items_sold: Record<string, number>;
    items_bought: Record<string, number>;
  };
  sse: {
    client_count: number;
  };
}

// User
export interface User {
  id: string;
  platform: string;
  platform_id: string;
  username: string;
  created_at: string;
}

export interface ActiveChatter {
  user_id: string;
  username: string;
  platform: string;
  last_message_at: string;
}

export interface InventoryItem {
  item_name: string;
  public_name?: string;
  quantity: number;
  quality_level?: string;
}

// Jobs
export interface UserJob {
  job_key: string;
  level: number;
  xp: number;
  xp_to_next: number;
}

// Stats
export interface UserStats {
  total_events: number;
  events_by_type: Record<string, number>;
  current_streak?: number;
}

// Progression
export interface VotingSession {
  id: string;
  status: string;
  tier: number;
  started_at: string;
  ended_at?: string;
  options?: VotingOption[];
}

export interface VotingOption {
  node_key: string;
  votes: number;
}

// Event log
export interface EventLogEntry {
  id: number;
  event_type: string;
  user_id?: string;
  payload: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  created_at: string;
}

// SSE events
export interface SSEEvent {
  id: string;
  type: string;
  timestamp: number;
  payload: unknown;
}

// Scenario
export interface ScenarioInfo {
  id: string;
  name: string;
  description: string;
  feature: string;
  step_count: number;
}

export interface ScenarioResult {
  scenario_id: string;
  scenario_name: string;
  success: boolean;
  duration_ms: number;
  steps: ScenarioStep[];
  error?: string;
  final_state?: Record<string, unknown>;
}

export interface ScenarioStep {
  step_name: string;
  step_index: number;
  success: boolean;
  duration_ms: number;
  output?: Record<string, unknown>;
  error?: string;
}

// Cache
export interface CacheStats {
  hits: number;
  misses: number;
  size: number;
  hit_rate: number;
}

// Quest
export interface QuestProgress {
  quest_id: string;
  quest_name: string;
  progress: number;
  target: number;
  completed: boolean;
}

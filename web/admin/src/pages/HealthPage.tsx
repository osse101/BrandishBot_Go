import { usePolling } from "../hooks/usePolling";
import { StatusBadge } from "../components/shared/StatusBadge";
import { formatNumber, formatMs } from "../utils/format";
import type {
  HealthResponse,
  VersionInfo,
  AdminMetrics,
  VelocityMetrics,
} from "../api/types";

export function HealthPage() {
  const health = usePolling<HealthResponse>("/healthz", 10000);
  const ready = usePolling<HealthResponse>("/readyz", 10000);
  const version = usePolling<VersionInfo>("/version", 60000);
  const metrics = usePolling<AdminMetrics>("/api/v1/admin/metrics", 5000);
  const velocity = usePolling<VelocityMetrics>(
    "/api/v1/progression/velocity?days=7",
    60000,
  );

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-gray-100">System Health</h2>

      {/* Server Status */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatusCard title="Liveness" loading={health.isLoading}>
          <StatusBadge
            status={
              health.data?.status === "ok"
                ? "ok"
                : health.error
                  ? "error"
                  : "unknown"
            }
            label={
              health.data?.status === "ok"
                ? "Healthy"
                : (health.error ?? "Checking...")
            }
          />
        </StatusCard>
        <StatusCard title="Readiness" loading={ready.isLoading}>
          <StatusBadge
            status={
              ready.data?.status === "ok"
                ? "ok"
                : ready.error
                  ? "error"
                  : "unknown"
            }
            label={
              ready.data?.status === "ok"
                ? "Ready"
                : (ready.error ?? "Checking...")
            }
          />
        </StatusCard>
        <StatusCard title="Build Info" loading={!version.data}>
          {version.data && (
            <div className="text-xs text-gray-400 space-y-1">
              <p>
                Version:{" "}
                <span className="text-gray-200">{version.data.version}</span>
              </p>
              <p>
                Go:{" "}
                <span className="text-gray-200">{version.data.go_version}</span>
              </p>
              <p>
                Commit:{" "}
                <span className="text-gray-200 font-mono">
                  {version.data.git_commit}
                </span>
              </p>
            </div>
          )}
        </StatusCard>
      </div>

      {/* Metrics Grid */}
      {metrics.data && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard
            label="In-Flight Requests"
            value={formatNumber(metrics.data.http.in_flight)}
          />
          <MetricCard
            label="Avg Latency"
            value={formatMs(metrics.data.http.avg_latency_ms)}
          />
          <MetricCard
            label="P95 Latency"
            value={formatMs(metrics.data.http.p95_latency_ms)}
          />
          <MetricCard
            label="SSE Clients"
            value={formatNumber(metrics.data.sse.client_count)}
          />
          <MetricCard
            label="Total Requests"
            value={formatNumber(
              sumValues(metrics.data.http.requests_total_by_status),
            )}
          />
          <MetricCard
            label="Error Requests"
            value={formatNumber(
              errorCount(metrics.data.http.requests_total_by_status),
            )}
          />
          <MetricCard
            label="Events Published"
            value={formatNumber(
              sumValues(metrics.data.events.published_total_by_type),
            )}
          />
          <MetricCard
            label="Event Errors"
            value={formatNumber(
              sumValues(metrics.data.events.handler_errors_by_type),
            )}
          />
        </div>
      )}

      {metrics.error && (
        <p className="text-sm text-red-400">
          Metrics unavailable: {metrics.error}
        </p>
      )}

      {/* Velocity Metrics */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">
          Progression Velocity (Last 7 Days)
        </h3>
        {velocity.data ? (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="flex flex-col">
              <span className="text-xs text-gray-500">Points / Day</span>
              <span className="text-lg font-semibold text-gray-100">
                {formatNumber(velocity.data.points_per_day)}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-gray-500">Trend</span>
              <span
                className={`text-lg font-semibold ${
                  velocity.data.trend === "increasing"
                    ? "text-green-400"
                    : velocity.data.trend === "decreasing"
                      ? "text-red-400"
                      : "text-gray-400"
                }`}
              >
                {velocity.data.trend.charAt(0).toUpperCase() +
                  velocity.data.trend.slice(1)}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-gray-500">Sample Size</span>
              <span className="text-lg font-semibold text-gray-100">
                {velocity.data.sample_size} days
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-gray-500">Total Points</span>
              <span className="text-lg font-semibold text-gray-100">
                {formatNumber(velocity.data.total_points)}
              </span>
            </div>
          </div>
        ) : velocity.error ? (
          <p className="text-xs text-red-400">
            Failed to load velocity: {velocity.error}
          </p>
        ) : (
          <p className="text-xs text-gray-500">Loading velocity metrics...</p>
        )}
      </div>

      {/* Business Metrics */}
      {metrics.data &&
        (Object.keys(metrics.data.business.items_sold).length > 0 ||
          Object.keys(metrics.data.business.items_bought).length > 0) && (
          <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
            <h3 className="text-sm font-medium text-gray-300 mb-3">
              Business Metrics
            </h3>
            <div className="grid grid-cols-2 gap-4">
              {Object.keys(metrics.data.business.items_sold).length > 0 && (
                <div>
                  <p className="text-xs text-gray-500 mb-1">Items Sold</p>
                  {Object.entries(metrics.data.business.items_sold).map(
                    ([item, count]) => (
                      <p key={item} className="text-xs text-gray-400">
                        {item}:{" "}
                        <span className="text-gray-200">
                          {formatNumber(count)}
                        </span>
                      </p>
                    ),
                  )}
                </div>
              )}
              {Object.keys(metrics.data.business.items_bought).length > 0 && (
                <div>
                  <p className="text-xs text-gray-500 mb-1">Items Bought</p>
                  {Object.entries(metrics.data.business.items_bought).map(
                    ([item, count]) => (
                      <p key={item} className="text-xs text-gray-400">
                        {item}:{" "}
                        <span className="text-gray-200">
                          {formatNumber(count)}
                        </span>
                      </p>
                    ),
                  )}
                </div>
              )}
            </div>
          </div>
        )}
    </div>
  );
}

function StatusCard({
  title,
  loading,
  children,
}: {
  title: string;
  loading: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
      <p className="text-xs text-gray-500 mb-2">{title}</p>
      {loading ? (
        <span className="text-xs text-gray-500">Loading...</span>
      ) : (
        children
      )}
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-lg font-semibold text-gray-100 mt-1">{value}</p>
    </div>
  );
}

function sumValues(record: Record<string, number>): number {
  return Object.values(record).reduce((a, b) => a + b, 0);
}

function errorCount(byStatus: Record<string, number>): number {
  return Object.entries(byStatus)
    .filter(([status]) => status.startsWith("4") || status.startsWith("5"))
    .reduce((sum, [, count]) => sum + count, 0);
}

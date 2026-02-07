import { useState, useEffect } from "react";
import { apiGet, apiPost } from "../api/client";
import { useToast } from "../components/shared/Toast";
import { JsonViewer } from "../components/shared/JsonViewer";
import { DataTable } from "../components/shared/DataTable";
import type {
  User,
  InventoryItem,
  UserJob,
  EventLogEntry,
  QuestProgress,
} from "../api/types";

export function UsersPage() {
  const toast = useToast();
  const [platform, setPlatform] = useState("twitch");
  const [username, setUsername] = useState("");
  const [searching, setSearching] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [activeTab, setActiveTab] = useState<
    "inventory" | "jobs" | "stats" | "quests" | "events"
  >("inventory");

  // Tab data
  const [inventory, setInventory] = useState<InventoryItem[]>([]);
  const [jobs, setJobs] = useState<UserJob[]>([]);
  const [stats, setStats] = useState<unknown>(null);
  const [quests, setQuests] = useState<QuestProgress[]>([]);
  const [events, setEvents] = useState<EventLogEntry[]>([]);
  const [allItems, setAllItems] = useState<
    { internal_name: string; public_name?: string }[]
  >([]);
  const [allJobs, setAllJobs] = useState<
    { Key: string; DisplayName: string }[]
  >([]);

  useEffect(() => {
    const fetchAutocompleteData = async () => {
      try {
        const items = await apiGet<
          { internal_name: string; public_name?: string }[]
        >("/api/v1/admin/items");
        setAllItems(items ?? []);
        const jobs =
          await apiGet<{ Key: string; DisplayName: string }[]>(
            "/api/v1/admin/jobs",
          );
        setAllJobs(jobs ?? []);
      } catch {
        // Silently fail
      }
    };
    fetchAutocompleteData();
  }, []);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) return;
    setSearching(true);
    setUser(null);
    try {
      const u = await apiGet<User>(
        `/api/v1/admin/users/lookup?platform=${platform}&username=${encodeURIComponent(username.trim())}`,
      );
      setUser(u);
      // Load initial tab data
      loadTabData("inventory", platform, username.trim());
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "User not found");
    } finally {
      setSearching(false);
    }
  };

  const loadTabData = async (tab: string, plat: string, uname: string) => {
    try {
      switch (tab) {
        case "inventory": {
          const inv = await apiGet<{ items: InventoryItem[] }>(
            `/api/v1/user/inventory-by-username?platform=${plat}&username=${encodeURIComponent(uname)}`,
          );
          setInventory(inv.items ?? []);
          break;
        }
        case "jobs": {
          const j = await apiGet<{ jobs: UserJob[] }>(
            `/api/v1/jobs/user?platform=${plat}&username=${encodeURIComponent(uname)}`,
          );
          setJobs(j.jobs ?? []);
          break;
        }
        case "stats": {
          const s = await apiGet<unknown>(
            `/api/v1/stats/user?platform=${plat}&username=${encodeURIComponent(uname)}`,
          );
          setStats(s);
          break;
        }
        case "quests": {
          if (user) {
            const q = await apiGet<QuestProgress[]>(
              `/api/v1/quests/progress?user_id=${user.id}`,
            );
            setQuests(q ?? []);
          }
          break;
        }
        case "events": {
          if (user) {
            const ev = await apiGet<{ events: EventLogEntry[] }>(
              `/api/v1/admin/events?user_id=${user.id}&limit=50`,
            );
            setEvents(ev.events ?? []);
          }
          break;
        }
      }
    } catch {
      // Silently handle — data just won't populate
    }
  };

  const switchTab = (tab: typeof activeTab) => {
    setActiveTab(tab);
    if (user) loadTabData(tab, platform, username.trim());
  };

  // Admin actions
  const [addItemForm, setAddItemForm] = useState({
    item_name: "",
    quantity: "1",
  });
  const [removeItemForm, setRemoveItemForm] = useState({
    item_name: "",
    quantity: "1",
  });
  const [xpForm, setXpForm] = useState({ job_key: "", amount: "" });

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-gray-100">User Management</h2>

      {/* Search */}
      <form onSubmit={handleSearch} className="flex items-end gap-2">
        <div>
          <label className="text-xs text-gray-500 block mb-1">Platform</label>
          <select
            value={platform}
            onChange={(e) => setPlatform(e.target.value)}
            className="px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
          >
            <option value="twitch">Twitch</option>
            <option value="discord">Discord</option>
            <option value="youtube">YouTube</option>
          </select>
        </div>
        <div className="flex-1">
          <label className="text-xs text-gray-500 block mb-1">Username</label>
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
            placeholder="Search username..."
          />
        </div>
        <button
          type="submit"
          disabled={searching || !username.trim()}
          className="px-4 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
        >
          {searching ? "Searching..." : "Search"}
        </button>
      </form>

      {/* Recently Active Users */}
      {!user && !searching && (
        <RecentUsersList
          onSelectUser={(u) => {
            setPlatform(u.platform);
            setUsername(u.username);
            setUser(u);
            loadTabData("inventory", u.platform, u.username);
          }}
        />
      )}

      {/* User Profile */}
      {user && (
        <div className="bg-gray-900 rounded-lg border border-gray-800">
          <div className="p-4 border-b border-gray-800">
            <div className="flex items-center gap-4">
              <div>
                <p className="text-lg font-medium text-gray-100">
                  {user.username}
                </p>
                <p className="text-xs text-gray-500">
                  {user.platform} | ID: {user.id}
                </p>
              </div>
            </div>
          </div>

          {/* Tabs */}
          <div className="flex border-b border-gray-800">
            {(["inventory", "jobs", "stats", "quests", "events"] as const).map(
              (tab) => (
                <button
                  key={tab}
                  onClick={() => switchTab(tab)}
                  className={`px-4 py-2 text-sm capitalize transition-colors ${
                    activeTab === tab
                      ? "text-blue-400 border-b-2 border-blue-400"
                      : "text-gray-400 hover:text-gray-200"
                  }`}
                >
                  {tab}
                </button>
              ),
            )}
          </div>

          {/* Tab content */}
          <div className="p-4">
            {activeTab === "inventory" && (
              <DataTable
                columns={[
                  { key: "item_name", header: "Item" },
                  { key: "public_name", header: "Display Name" },
                  { key: "quantity", header: "Qty" },
                  { key: "shine_level", header: "Shine" },
                ]}
                data={inventory as unknown as Record<string, unknown>[]}
                keyField="item_name"
                emptyMessage="No items in inventory"
              />
            )}

            {activeTab === "jobs" && (
              <DataTable
                columns={[
                  { key: "job_key", header: "Job" },
                  { key: "level", header: "Level" },
                  { key: "xp", header: "XP" },
                  { key: "xp_to_next", header: "XP to Next" },
                ]}
                data={jobs as unknown as Record<string, unknown>[]}
                keyField="job_key"
                emptyMessage="No job data"
              />
            )}

            {activeTab === "stats" &&
              (stats ? (
                <JsonViewer data={stats} defaultExpanded />
              ) : (
                <p className="text-sm text-gray-500">No stats</p>
              ))}

            {activeTab === "quests" && (
              <DataTable
                columns={[
                  { key: "quest_name", header: "Quest" },
                  { key: "progress", header: "Progress" },
                  { key: "target", header: "Target" },
                  {
                    key: "completed",
                    header: "Done",
                    render: (r) =>
                      (r as Record<string, unknown>).completed ? "Yes" : "No",
                  },
                ]}
                data={quests as unknown as Record<string, unknown>[]}
                keyField="quest_id"
                emptyMessage="No active quests"
              />
            )}

            {activeTab === "events" && (
              <div className="space-y-1 max-h-96 overflow-y-auto">
                {events.length === 0 && (
                  <p className="text-sm text-gray-500">No events</p>
                )}
                {events.map((evt) => (
                  <div
                    key={evt.id}
                    className="text-xs border-b border-gray-800 py-2"
                  >
                    <span className="text-gray-500 font-mono mr-2">
                      {new Date(evt.created_at).toLocaleString()}
                    </span>
                    <span className="text-blue-400 mr-2">{evt.event_type}</span>
                    <JsonViewer data={evt.payload} />
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Admin Actions */}
      {user && (
        <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
          <h3 className="text-sm font-medium text-gray-300 mb-3">
            Admin Actions
          </h3>

          {/* Add Item */}
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              try {
                await apiPost("/api/v1/user/item/add", {
                  platform,
                  username: username.trim(),
                  item_name: addItemForm.item_name,
                  quantity: Number(addItemForm.quantity),
                });
                toast.success(
                  `Added ${addItemForm.quantity}x ${addItemForm.item_name}`,
                );
                setAddItemForm({ item_name: "", quantity: "1" });
                loadTabData("inventory", platform, username.trim());
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed");
              }
            }}
            className="flex items-end gap-2 mb-3"
          >
            <div className="flex-1">
              <label className="text-xs text-gray-500 block mb-1">
                Add Item
              </label>
              <input
                value={addItemForm.item_name}
                onChange={(e) =>
                  setAddItemForm((f) => ({ ...f, item_name: e.target.value }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                placeholder="item name"
                list="all-items"
              />
              <datalist id="all-items">
                {allItems.map((it) => (
                  <option key={it.internal_name} value={it.internal_name}>
                    {it.public_name}
                  </option>
                ))}
              </datalist>
            </div>
            <div className="w-20">
              <label className="text-xs text-gray-500 block mb-1">Qty</label>
              <input
                type="number"
                min="1"
                value={addItemForm.quantity}
                onChange={(e) =>
                  setAddItemForm((f) => ({ ...f, quantity: e.target.value }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              />
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 text-sm rounded-md bg-green-600 text-white hover:bg-green-500 transition-colors"
            >
              Add
            </button>
          </form>

          {/* Remove Item */}
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              try {
                await apiPost("/api/v1/user/item/remove", {
                  platform,
                  username: username.trim(),
                  item_name: removeItemForm.item_name,
                  quantity: Number(removeItemForm.quantity),
                });
                toast.success(
                  `Removed ${removeItemForm.quantity}x ${removeItemForm.item_name}`,
                );
                setRemoveItemForm({ item_name: "", quantity: "1" });
                loadTabData("inventory", platform, username.trim());
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed");
              }
            }}
            className="flex items-end gap-2 mb-3"
          >
            <div className="flex-1">
              <label className="text-xs text-gray-500 block mb-1">
                Remove Item
              </label>
              <input
                value={removeItemForm.item_name}
                onChange={(e) =>
                  setRemoveItemForm((f) => ({
                    ...f,
                    item_name: e.target.value,
                  }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                placeholder="item name"
                list="all-items"
              />
            </div>
            <div className="w-20">
              <label className="text-xs text-gray-500 block mb-1">Qty</label>
              <input
                type="number"
                min="1"
                value={removeItemForm.quantity}
                onChange={(e) =>
                  setRemoveItemForm((f) => ({ ...f, quantity: e.target.value }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              />
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 text-sm rounded-md bg-red-600 text-white hover:bg-red-500 transition-colors"
            >
              Remove
            </button>
          </form>

          {/* Award XP */}
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              try {
                await apiPost("/api/v1/admin/jobs/award-xp", {
                  platform,
                  username: username.trim(),
                  job_key: xpForm.job_key,
                  amount: Number(xpForm.amount),
                });
                toast.success(`Awarded ${xpForm.amount} XP`);
                setXpForm({ job_key: "", amount: "" });
                loadTabData("jobs", platform, username.trim());
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed");
              }
            }}
            className="flex items-end gap-2 mb-3"
          >
            <div className="flex-1">
              <label className="text-xs text-gray-500 block mb-1">
                Award XP (Job Key)
              </label>
              <input
                value={xpForm.job_key}
                onChange={(e) =>
                  setXpForm((f) => ({ ...f, job_key: e.target.value }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                placeholder="e.g. blacksmith"
                list="all-jobs"
              />
              <datalist id="all-jobs">
                {allJobs.map((j) => (
                  <option key={j.Key} value={j.Key}>
                    {j.DisplayName}
                  </option>
                ))}
              </datalist>
            </div>
            <div className="w-20">
              <label className="text-xs text-gray-500 block mb-1">Amount</label>
              <input
                type="number"
                value={xpForm.amount}
                onChange={(e) =>
                  setXpForm((f) => ({ ...f, amount: e.target.value }))
                }
                className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
                placeholder="100"
              />
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 transition-colors"
            >
              Award XP
            </button>
          </form>

          {/* Clear Timeout */}
          <button
            onClick={async () => {
              try {
                await apiPost("/api/v1/admin/timeout/clear", {
                  platform,
                  username: username.trim(),
                });
                toast.success("Timeout cleared");
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed");
              }
            }}
            className="px-3 py-1.5 text-sm rounded-md bg-yellow-600 text-white hover:bg-yellow-500 transition-colors"
          >
            Clear Timeout
          </button>
        </div>
      )}
    </div>
  );
}

function RecentUsersList({
  onSelectUser,
}: {
  onSelectUser: (u: User) => void;
}) {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchRecent = async () => {
    try {
      const data = await apiGet<User[]>("/api/v1/admin/users/recent");
      setUsers(data ?? []);
    } catch {
      // Handle error
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRecent();
  }, []);

  if (loading)
    return <p className="text-sm text-gray-500">Loading recent users...</p>;
  if (users.length === 0) return null;

  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
      <h3 className="text-sm font-medium text-gray-300 mb-3">
        Recently Active Users
      </h3>
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
        {users.map((u) => (
          <button
            key={u.id}
            onClick={() => onSelectUser(u)}
            className="text-left p-3 rounded-md bg-gray-800 border border-gray-700 hover:border-blue-500 transition-colors group"
          >
            <div className="text-sm font-medium text-gray-200 group-hover:text-blue-400">
              {u.username}
            </div>
            <div className="text-xs text-gray-500">{u.platform}</div>
          </button>
        ))}
      </div>
    </div>
  );
}

# 🛠️ AGENTS & SERVICES Overview

This document describes the core agents, services, and communication patterns within the system. The architecture relies heavily on an **Event-Driven Architecture (EDA)** where services communicate asynchronously via an Event Broker.

## Core Communication Pattern: Event-Driven Architecture (EDA)

The system utilizes an **Event Broker** as the central message bus. Services publish **Events** (simple struct messages) to the broker, and other services act as **Event Handlers** by subscribing to relevant events and executing their business logic.

This pattern ensures **decoupling** between services, allowing them to operate independently and scale separately.

| Component | Type | Primary Function | Communication |
| :--- | :--- | :--- | :--- |
| **Main Application** | REST API | Handles user requests, authentication, and core transactional logic (e.g., Inventory updates). | **Inbound:** REST/HTTP |
| **Event Broker** | Message Bus | Receives, queues, and broadcasts Events to all registered Handlers. | Internal (Go interfaces/package) |
| **Inventory Service** | Event Publisher/Handler | Manages item ownership, validates transactions, and publishes inventory-related events. | REST (via Main App), Events (Outbound) |
| **Stats Service** | Event Handler | Listens for key events to update user statistics (e.g., counts of actions taken). | Events (Inbound) |
| **Class Service** | Service/Logic | Allocates experience points (XP) and computes the effects and power levels of in-game classes/abilities. | REST (via Main App/Other Services), Events (Inbound/Outbound) |

---

## 🔄 Detailed Agent Flows

### 1. The Transactional Flow (REST + Event Publishing)

The main application handles synchronous, critical updates using traditional **REST** calls, followed immediately by an event publication.

| Step | Agent | Action | Communication |
| :--- | :--- | :--- | :--- |
| 1. | **User/Client** | Initiates item transfer. | REST (Main Application) |
| 2. | **Main Application** | Calls `Inventory Service` to execute transfer logic. | REST |
| 3. | **Inventory Service** | **Updates Database** & publishes event. | **Publishes `ItemGivenEvent`** |

### 2. The Asynchronous Reaction Flow (Event Handling)

This flow illustrates how decoupled agents react to events without direct knowledge of the publisher.

| Step | Event | Publisher | Handlers | Handler Action |
| :--- | :--- | :--- | :--- | :--- |
| 1. | `ItemGivenEvent` | **Inventory Service** | **Stats Service** | Increments `items_given` and `items_received` counts in the database. |
| 2. | `UserJoinedEvent` | *Example Publisher* | **Class Service** | Allocates initial XP/starting class to the new user. |
| 3. | `ItemUsedEvent` | **Inventory Service** | **Class Service** | Computes if item use grants bonus XP or affects class abilities. |

---

## 🏗️ Go Implementation Notes

### Event Broker

The `Event Broker` should be implemented as a lightweight Go package or interface within the project, likely utilizing **concurrent maps** or **channels** to manage handler subscriptions and safely dispatch events to all subscribers.

```go
// EventBroker Interface Sketch
type EventBroker interface {
    // Publish sends an event to all subscribed handlers
    Publish(event Event)
    // Subscribe registers a handler function for a specific event type
    Subscribe(eventType string, handler func(event Event))
}
```

## Future Considerations

As this system grows, we will incrementally add bounded contexts:

- **Stats Service** – Listens to inventory events and recomputes user statistics asynchronously.
- **Class Service** – Listens to XP-gain events and triggers ability unlocks or level-ups.

Each service will consume an event stream published by the core inventory system, keeping them loosely coupled yet consistent.

---

## 🤖 AI Agent Best Practices

### Process Management & Cleanup

When testing the application, AI agents should follow these cleanup practices:

**Starting Background Processes:**

```powershell
# Background commands return a command ID
go run cmd/app/main.go
# Returns: Background command ID: abc123-def456-...
```

**Tracking Command IDs:**

- **ALWAYS** store the command ID returned from `run_command` when starting background processes
- Use this ID for targeted cleanup instead of searching by port/name
- Track IDs in a list throughout the session

**Cleanup Process:**

```powershell
# ✅ CORRECT: Use the tracked command ID
send_command_input(CommandId: "abc123-def456-...", Terminate: true)

# ❌ AVOID: Searching for processes by port (unreliable, can kill wrong processes)
# Get-NetTCPConnection -LocalPort 8080 | ... | Stop-Process
```

**End-of-Session Cleanup:**

- **MANDATORY**: Terminate ALL background processes at the end of testing
- Track all started command IDs during the session
- Clean up in reverse order (newest to oldest)
- Verify cleanup success with `command_status` tool

**Example Workflow:**

```powershell
1. Start server → Track: cmd_id_server
2. Run tests
3. Start debug script → Track: cmd_id_debug
4. Verify results
5. Cleanup: send_command_input(cmd_id_debug, Terminate=true)
6. Cleanup: send_command_input(cmd_id_server, Terminate=true)
7. Verify: command_status for both IDs shows "DONE"
```

**Why This Matters:**

- Prevents resource leaks
- Avoids port conflicts in future sessions
- Ensures clean state for next agent interaction
- More reliable than port-based process killing

**Key principle**: Log errors at the boundary where they occur, with full context.

### 6. Verify Fixes

After implementing a fix:

1. **Rebuild** the application
2. **Re-run** the reproduction script
3. **Check logs** for successful execution
4. **Verify** expected behavior matches actual outcome

### 7. Document Findings

Update relevant documentation:

- Fix schema mismatches in migration files or code
- Add comments explaining non-obvious error handling
- Update ARCHITECTURE.md if design assumptions were wrong

---

**Real Example Summary (lootbox1):**

| Attempt | Error Discovered | Fix Applied |
|---------|------------------|-------------|
| 1 | SQL column `p.platform_name` doesn't exist | Changed to `p.name` in queries |
| 2 | Logic error: "user not found" treated as fatal | Changed to check error message and continue |
| 3 | Missing platform data: "no rows in result set" | Identified need to seed platforms table |

**Result**: DEBUG logs provided complete visibility into the data flow, enabling rapid root cause identification.

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
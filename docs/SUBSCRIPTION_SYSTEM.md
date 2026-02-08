# Subscription System

The subscription system tracks Twitch subscriptions (Tier 1/2/3) and YouTube memberships, providing a foundation for subscription-based features and rewards.

## Architecture Overview

```
Streamer.bot → Webhook → Service → Database
                   ↓
               Event Bus → SSE/Streamer.bot

Background Worker → Check Expired → Verify → Update
```

### Components

- **Database**: 3 tables (`subscription_tiers`, `user_subscriptions`, `subscription_history`)
- **Service**: Handles webhook events, verification requests, cached status checks
- **Worker**: Checks for expired subscriptions every 6 hours, requests verification
- **Cache**: 5-minute TTL in-memory cache for subscription status lookups
- **Events**: Publishes lifecycle events to event bus (activated, renewed, expired, etc.)

---

## Checking Subscription Status

### Quick Reference

```go
// Check if user is subscribed
isSubscribed, err := subscriptionService.IsSubscribed(ctx, userID, "twitch")

// Get subscription tier details
tierName, tierLevel, err := subscriptionService.GetSubscriptionTier(ctx, userID, "twitch")
```

### Using in Your Service

**Example: Progression Service with Subscription Bonuses**

```go
type service struct {
    repo               repository.Progression
    subscriptionSvc    subscription.Service  // Inject subscription service
    // ... other fields
}

func NewService(
    repo repository.Progression,
    subscriptionSvc subscription.Service,
    // ... other params
) Service {
    return &service{
        repo:            repo,
        subscriptionSvc: subscriptionSvc,
        // ...
    }
}

// Apply subscription bonus to engagement points
func (s *service) RecordEngagement(ctx context.Context, userID, source string, basePoints int) error {
    points := basePoints

    // Check if user is subscribed (cached, fast)
    isSubscribed, err := s.subscriptionSvc.IsSubscribed(ctx, userID, "twitch")
    if err != nil {
        slog.Warn("Failed to check subscription status", "error", err)
        // Continue with base points if check fails
    } else if isSubscribed {
        // Get tier for bonus calculation
        tierName, tierLevel, _ := s.subscriptionSvc.GetSubscriptionTier(ctx, userID, "twitch")

        // Apply tier-based bonus
        bonus := 1.0 + (float64(tierLevel) * 0.1) // 10% per tier level
        points = int(float64(points) * bonus)

        slog.Debug("Subscription bonus applied",
            "user_id", userID,
            "tier", tierName,
            "bonus", bonus,
            "base_points", basePoints,
            "final_points", points)
    }

    return s.repo.AddEngagement(ctx, userID, source, points)
}
```

**Example: Economy Service with Subscriber Discounts**

```go
func (s *service) BuyItem(ctx context.Context, userID, itemName string, quantity int) error {
    item, err := s.itemRepo.GetItemByName(ctx, itemName)
    if err != nil {
        return err
    }

    basePrice := item.Price * quantity

    // Check subscription for discount
    isSubscribed, _ := s.subscriptionSvc.IsSubscribed(ctx, userID, "twitch")
    finalPrice := basePrice
    if isSubscribed {
        tierName, tierLevel, _ := s.subscriptionSvc.GetSubscriptionTier(ctx, userID, "twitch")
        discount := float64(tierLevel) * 0.05 // 5% per tier
        finalPrice = int(float64(basePrice) * (1.0 - discount))

        slog.Info("Subscriber discount applied",
            "user_id", userID,
            "tier", tierName,
            "discount_percent", discount*100,
            "base_price", basePrice,
            "final_price", finalPrice)
    }

    // Deduct money and add item
    return s.processTransaction(ctx, userID, finalPrice, itemName, quantity)
}
```

### Performance Characteristics

- **First call**: ~1-5ms (database query)
- **Cached calls**: ~0.01ms (in-memory lookup)
- **Cache TTL**: 5 minutes
- **Cache invalidation**: Automatic on subscription changes

### Error Handling

The `IsSubscribed()` and `GetSubscriptionTier()` methods are designed to **fail gracefully**:

```go
isSubscribed, err := subscriptionSvc.IsSubscribed(ctx, userID, platform)
if err != nil {
    // Log but don't fail - assume not subscribed
    slog.Warn("Subscription check failed", "error", err)
    isSubscribed = false
}

// Continue with logic
if isSubscribed {
    // Apply bonus
}
```

**Important**: These methods return `(false, nil)` if the user is not subscribed. Only database errors return non-nil errors.

---

## Subscription Lifecycle

### 1. New Subscription

**Flow**:
1. User subscribes on Twitch/YouTube
2. Streamer.bot detects event → calls webhook
3. BrandishBot receives event, creates subscription record (30-day expiry)
4. Publishes `subscription.activated` event
5. Cache is empty (will be populated on first check)

**Webhook Payload**:
```json
{
  "platform": "twitch",
  "platform_user_id": "12345678",
  "username": "viewer123",
  "tier_name": "tier1",
  "event_type": "subscribed",
  "timestamp": 1234567890
}
```

### 2. Renewal

**Flow**:
1. User renews subscription (or auto-renews)
2. Streamer.bot calls webhook with `event_type: "renewed"`
3. BrandishBot updates expiration date (+30 days)
4. Publishes `subscription.renewed` event
5. Cache is invalidated for this user

### 3. Upgrade/Downgrade

**Flow**:
1. User changes tier (e.g., Tier 1 → Tier 2)
2. Streamer.bot calls webhook with `event_type: "upgraded"`
3. BrandishBot updates `tier_id` and extends expiration
4. Publishes `subscription.upgraded` event
5. Cache is invalidated

### 4. Expiration

**Flow**:
1. Background worker runs (every 6 hours)
2. Finds subscriptions where `expires_at < now` AND `status = 'active'`
3. Marks subscription as `expired` in database
4. Requests verification from Streamer.bot (async)
5. Streamer.bot queries platform API for current status
6. Streamer.bot calls webhook with current status
7. If renewed: status updated to `active`, expiration extended
8. If still expired: remains `expired`
9. Cache is invalidated

**Why check after expiry?**
- Verifies actual platform state, not predictions
- Handles auto-renewals that may occur at expiration time
- Provides grace period for Streamer.bot to detect renewals
- Prevents false negatives from timezone/timing issues

### 5. Cancellation

**Flow**:
1. User cancels subscription (doesn't auto-renew)
2. Streamer.bot calls webhook with `event_type: "cancelled"`
3. BrandishBot marks status as `cancelled`
4. Subscription remains valid until `expires_at`
5. After expiration, worker marks as `expired`

---

## Database Schema

### subscription_tiers (Reference Data)

```sql
CREATE TABLE subscription_tiers (
    tier_id      SERIAL PRIMARY KEY,
    platform     VARCHAR(50) NOT NULL,
    tier_name    VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    tier_level   INT NOT NULL,
    UNIQUE (platform, tier_name)
);

-- Seed data: twitch.tier1, twitch.tier2, twitch.tier3, youtube.member
```

### user_subscriptions (Current Status)

```sql
CREATE TABLE user_subscriptions (
    user_id          UUID NOT NULL,
    platform         VARCHAR(50) NOT NULL,
    tier_id          INT NOT NULL,
    status           VARCHAR(20) NOT NULL, -- 'active', 'expired', 'cancelled'
    subscribed_at    TIMESTAMPTZ NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    last_verified_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, platform)
);

-- Index for expiration checks
CREATE INDEX idx_user_subscriptions_expiring ON user_subscriptions(expires_at)
    WHERE status = 'active';
```

### subscription_history (Audit Trail)

```sql
CREATE TABLE subscription_history (
    history_id    BIGSERIAL PRIMARY KEY,
    user_id       UUID NOT NULL,
    platform      VARCHAR(50) NOT NULL,
    tier_id       INT NOT NULL,
    event_type    VARCHAR(50) NOT NULL, -- 'subscribed', 'renewed', 'upgraded', etc.
    subscribed_at TIMESTAMPTZ NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Configuration

### Environment Variables

```bash
# Worker Settings
SUBSCRIPTION_CHECK_INTERVAL=6h      # How often to check for expired subscriptions
SUBSCRIPTION_DEFAULT_DURATION=720h  # 30 days - subscription length
SUBSCRIPTION_GRACE_PERIOD=24h       # (Unused - kept for compatibility)
```

### Cache Configuration

Cache TTL is hard-coded in the service:

```go
// internal/subscription/service.go
cache: NewStatusCache(5 * time.Minute)
```

To adjust, modify this value in `NewService()`.

---

## Event Types

The subscription service publishes the following events to the event bus:

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `subscription.activated` | New subscription | `{user_id, platform, tier_name, timestamp}` |
| `subscription.renewed` | Subscription renewed | `{user_id, platform, tier_name, timestamp}` |
| `subscription.upgraded` | Tier upgrade | `{user_id, platform, tier_name, timestamp}` |
| `subscription.downgraded` | Tier downgrade | `{user_id, platform, tier_name, timestamp}` |
| `subscription.expired` | Subscription expired | `{user_id, platform, tier_name, timestamp}` |
| `subscription.cancelled` | Subscription cancelled | `{user_id, platform, tier_name, timestamp}` |

### Subscribing to Events

```go
// In your service initialization
eventBus.Subscribe(event.SubscriptionActivated, s.handleSubscriptionActivated)

func (s *service) handleSubscriptionActivated(ctx context.Context, evt event.Event) error {
    payload := evt.Payload.(event.SubscriptionPayloadV1)

    slog.Info("User subscribed!",
        "user_id", payload.UserID,
        "platform", payload.Platform,
        "tier", payload.TierName)

    // Award welcome bonus, send notification, etc.
    return nil
}
```

---

## Streamer.bot Integration

### Required C# Actions

**1. Subscription Event Handler**

Calls BrandishBot webhook when subscription events occur:

```csharp
// Action: OnTwitchSubscription (or OnYouTubeMembership)
var payload = new {
    platform = "twitch",
    platform_user_id = args["userId"],
    username = args["userName"],
    tier_name = args["tier"], // "tier1", "tier2", "tier3"
    event_type = "subscribed", // or "renewed", "upgraded", "cancelled"
    timestamp = DateTimeOffset.Now.ToUnixTimeSeconds()
};

await CPH.SendHttpRequest("POST",
    "http://localhost:8080/api/v1/subscriptions/event",
    payload,
    headers: new Dictionary<string, string> {
        { "X-API-Key", apiKey },
        { "Content-Type", "application/json" }
    });
```

**2. Verification Action**

Called by BrandishBot worker to verify subscription status:

```csharp
// Action: BrandishBot_VerifySubscription
var platform = args["platform"];
var platformUserId = args["platform_user_id"];

// Query platform API for current subscription status
var isSubscribed = await TwitchSubscriptionStatus(platformUserId);
var tierName = isSubscribed ? GetCurrentTier(platformUserId) : null;

// Call back to BrandishBot with verification result
var payload = new {
    platform = platform,
    platform_user_id = platformUserId,
    username = GetUsername(platformUserId),
    tier_name = tierName,
    event_type = isSubscribed ? "renewed" : "cancelled",
    timestamp = DateTimeOffset.Now.ToUnixTimeSeconds()
};

await CPH.SendHttpRequest("POST",
    "http://localhost:8080/api/v1/subscriptions/event",
    payload,
    headers: headers);
```

---

## Testing

### Manual Testing

**1. Simulate Subscription Event**

```bash
curl -X POST http://localhost:8080/api/v1/subscriptions/event \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: your_api_key' \
  -d '{
    "platform": "twitch",
    "platform_user_id": "12345",
    "username": "testuser",
    "tier_name": "tier1",
    "event_type": "subscribed",
    "timestamp": 1234567890
  }'
```

**2. Verify Database**

```sql
SELECT * FROM user_subscriptions WHERE platform = 'twitch';
SELECT * FROM subscription_history ORDER BY created_at DESC LIMIT 10;
```

**3. Test Service Check**

```go
isSubscribed, err := subscriptionService.IsSubscribed(ctx, userID, "twitch")
fmt.Printf("Subscribed: %v, Error: %v\n", isSubscribed, err)
```

---

## Common Patterns

### Subscriber-Only Features

```go
func (s *service) UseSubscriberFeature(ctx context.Context, userID string) error {
    isSubscribed, err := s.subscriptionSvc.IsSubscribed(ctx, userID, "twitch")
    if err != nil {
        return fmt.Errorf("failed to check subscription: %w", err)
    }

    if !isSubscribed {
        return fmt.Errorf("this feature requires an active subscription")
    }

    // Feature logic here
    return s.executeFeature(ctx, userID)
}
```

### Tier-Based Rewards

```go
func (s *service) CalculateReward(ctx context.Context, userID string, baseReward int) int {
    tierName, tierLevel, err := s.subscriptionSvc.GetSubscriptionTier(ctx, userID, "twitch")
    if err != nil || tierLevel == 0 {
        return baseReward // Not subscribed or error
    }

    multiplier := map[string]float64{
        "tier1": 1.1,  // 10% bonus
        "tier2": 1.25, // 25% bonus
        "tier3": 1.5,  // 50% bonus
    }[tierName]

    return int(float64(baseReward) * multiplier)
}
```

### Subscriber Analytics

```go
func (s *service) GetSubscriberStats(ctx context.Context) (*SubscriberStats, error) {
    // Query all active subscriptions
    rows, err := s.db.Query(`
        SELECT platform, tier_name, COUNT(*)
        FROM user_subscriptions us
        JOIN subscription_tiers st ON us.tier_id = st.tier_id
        WHERE status = 'active'
        GROUP BY platform, tier_name
    `)
    // Process results...
}
```

---

## Troubleshooting

### Subscription not detected

1. Check Streamer.bot is calling the webhook
2. Verify API key in webhook request
3. Check BrandishBot logs for errors
4. Verify user exists in database
5. Check `subscription_history` for event records

### Cache issues

```go
// Force cache refresh
subscriptionService.cache.Invalidate(userID, platform)

// Or clear entire cache
subscriptionService.cache.InvalidateAll()
```

### Worker not running

```bash
# Check logs for worker startup
grep "Subscription worker" logs/brandishbot.log

# Verify configuration
echo $SUBSCRIPTION_CHECK_INTERVAL
```

---

## Future Enhancements

Potential extensions (not currently implemented):

- Multi-month subscriptions (adjust expiration based on duration)
- YouTube multi-tier support (if platform adds tiers)
- Subscription benefits system (configurable rewards per tier)
- Expiration warnings (7-day notice before expiry)
- Admin endpoints (manual subscription grants)
- Analytics dashboard (churn rate, tier distribution)

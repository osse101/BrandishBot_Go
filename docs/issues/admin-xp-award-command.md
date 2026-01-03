# Admin XP Award Command

**Priority:** HIGH  
**Complexity:** 4/10  
**Estimated Effort:** 2-3 hours  
**Created:** 2026-01-03

## Problem

There's currently no way for administrators to manually award job XP to users. This is needed for:

1. **Bug Compensation:** When users are affected by bugs (e.g., lost XP due to server issues)
2. **Event Rewards:** Community events, contests, giveaways
3. **Testing:** Testing the event system and level-up notifications
4. **Community Management:** Special recognition, milestones, partnerships

Without this feature, admins must directly modify the database (risky and time-consuming).

## Proposed Solution

Add an admin-only Discord command `/admin-award-xp` that allows administrators to award XP to any user's job.

**Command:**
```
/admin-award-xp @user job:explorer amount:1000 reason:"Bug compensation"
```

**Features:**
- Admin role permission check
- Audit logging (all grants recorded)
- Rate limiting (10 grants/minute)
- Triggers level-up events
- Backend API endpoint for flexibility

## Implementation

### 1. Backend API Endpoint

Create [`internal/handler/admin_job.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/admin_job.go):

```go
package handler

import (
    "encoding/json"
    "net/http"
    "github.com/osse101/BrandishBot_Go/internal/job"
    "github.com/osse101/BrandishBot_Go/internal/logger"
    "github.com/osse101/BrandishBot_Go/internal/audit"
)

// POST /api/v1/admin/award-xp
// Body: {"user_id": "uuid", "job_key": "explorer", "amount": 1000, "reason": "Bug compensation"}
func HandleAdminAwardXP(w http.ResponseWriter, r *http.Request, jobService job.Service, auditLogger *audit.Logger) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req struct {
        UserID  string `json:"user_id"`
        JobKey  string `json:"job_key"`
        Amount  int    `json:"amount"`
        Reason  string `json:"reason"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validation
    if req.UserID == "" || req.JobKey == "" || req.Amount <= 0 {
        http.Error(w, "user_id, job_key, and positive amount required", http.StatusBadRequest)
        return
    }
    
    if req.Amount > 10000 {
        http.Error(w, "amount exceeds maximum (10000)", http.StatusBadRequest)
        return
    }
    
    log := logger.FromContext(r.Context())
    log.Info("Admin XP award requested",
        "user_id", req.UserID,
        "job_key", req.JobKey,
        "amount", req.Amount,
        "reason", req.Reason)
    
    // Award XP
    result, err := jobService.AwardXP(
        r.Context(),
        req.UserID,
        req.JobKey,
        req.Amount,
        "admin_grant",
        map[string]interface{}{
            "reason":     req.Reason,
            "granted_by": r.Header.Get("X-Admin-ID"),
        },
    )
    
    if err != nil {
        log.Error("Failed to award XP", "error", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Audit log
    auditLogger.Log(audit.AuditEvent{
        Action:   "admin_award_xp",
        AdminID:  r.Header.Get("X-Admin-ID"),
        TargetID: req.UserID,
        Details: map[string]interface{}{
            "job_key": req.JobKey,
            "amount":  req.Amount,
            "reason":  req.Reason,
            "leveled_up": result.LeveledUp,
            "new_level": result.NewLevel,
        },
    })
    
    response := map[string]interface{}{
        "success": true,
        "result":  result,
        "message": "XP awarded successfully",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 2. Audit Logger

Create [`internal/audit/logger.go`](file:///home/osse1/projects/BrandishBot_Go/internal/audit/logger.go):

```go
package audit

import (
    "encoding/json"
    "os"
    "sync"
    "time"
)

type Logger struct {
    file *os.File
    mu   sync.Mutex
}

type AuditEvent struct {
    Timestamp time.Time              `json:"timestamp"`
    Action    string                 `json:"action"`
    AdminID   string                 `json:"admin_id"`
    TargetID  string                 `json:"target_id"`
    Details   map[string]interface{} `json:"details"`
}

func NewLogger(path string) (*Logger, error) {
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    return &Logger{file: f}, nil
}

func (al *Logger) Log(event AuditEvent) error {
    al.mu.Lock()
    defer al.mu.Unlock()
    
    event.Timestamp = time.Now()
    data, _ := json.Marshal(event)
    _, err := al.file.Write(append(data, '\n'))
    return err
}

func (al *Logger) Close() error {
    return al.file.Close()
}
```

### 3. Discord Command

Create [`internal/discord/cmd_admin.go`](file:///home/osse1/projects/BrandishBot_Go/internal/discord/cmd_admin.go):

```go
package discord

import (
    "fmt"
    "github.com/bwmarrin/discordgo"
)

var AdminAwardXPCommand = &discordgo.ApplicationCommand{
    Name:        "admin-award-xp",
    Description: "Award XP to a user's job (Admin only)",
    Options: []*discordgo.ApplicationCommandOption{
        {
            Type:        discordgo.ApplicationCommandOptionUser,
            Name:        "user",
            Description: "User to award XP to",
            Required:    true,
        },
        {
            Type:        discordgo.ApplicationCommandOptionString,
            Name:        "job",
            Description: "Job to award XP to",
            Required:    true,
            Choices: []*discordgo.ApplicationCommandOptionChoice{
                {Name: "Explorer", Value: "explorer"},
                {Name: "Blacksmith", Value: "blacksmith"},
            },
        },
        {
            Type:        discordgo.ApplicationCommandOptionInteger,
            Name:        "amount",
            Description: "Amount of XP to award",
            Required:    true,
            MinValue:    ptr(float64(1)),
            MaxValue:    ptr(float64(10000)),
        },
        {
            Type:        discordgo.ApplicationCommandOptionString,
            Name:        "reason",
            Description: "Reason for awarding XP",
            Required:    false,
        },
    },
}

func (h *CommandHandler) HandleAdminAwardXP(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Permission check
    if !isAdmin(i.Member) {
        respondError(s, i, "❌ This command requires administrator permissions")
        return
    }
    
    options := parseOptions(i.ApplicationCommandData().Options)
    
    targetUser := options["user"].UserValue(s)
    jobKey := options["job"].StringValue()
    amount := int(options["amount"].IntValue())
    reason := ""
    if r, ok := options["reason"]; ok {
        reason = r.StringValue()
    }
    
    // Call API
    resp, err := h.apiClient.AwardXP(targetUser.Username, jobKey, amount, reason)
    if err != nil {
        respondError(s, i, fmt.Sprintf("Failed to award XP: %v", err))
        return
    }
    
    message := fmt.Sprintf("✅ Awarded %d XP to %s's %s job",
        amount, targetUser.Mention(), jobKey)
    
    if resp.LeveledUp {
        message += fmt.Sprintf("\n🎉 %s leveled up to level %d!",
            targetUser.Mention(), resp.NewLevel)
    }
    
    if reason != "" {
        message += fmt.Sprintf("\n📝 Reason: %s", reason)
    }
    
    respondSuccess(s, i, message)
}

func isAdmin(member *discordgo.Member) bool {
    adminRoleID := os.Getenv("ADMIN_ROLE_ID")
    if adminRoleID == "" {
        return false
    }
    
    for _, roleID := range member.Roles {
        if roleID == adminRoleID {
            return true
        }
    }
    return false
}

func ptr[T any](v T) *T {
    return &v
}
```

### 4. API Client Method

Add to [`internal/discord/client.go`](file:///home/osse1/projects/BrandishBot_Go/internal/discord/client.go):

```go
type XPAwardResponse struct {
    Success   bool `json:"success"`
    LeveledUp bool `json:"leveled_up"`
    NewLevel  int  `json:"new_level"`
    NewXP     int  `json:"new_xp"`
}

func (c *APIClient) AwardXP(username, jobKey string, amount int, reason string) (*XPAwardResponse, error) {
    // First, get user ID by username
    user, err := c.getUserByUsername(username)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    payload := map[string]interface{}{
        "user_id": user.ID,
        "job_key": jobKey,
        "amount":  amount,
        "reason":  reason,
    }
    
    body, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", c.baseURL+"/api/v1/admin/award-xp", bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Admin-ID", "discord-bot")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API error: %s", string(body))
    }
    
    var result struct {
        Success bool `json:"success"`
        Result  struct {
            LeveledUp bool `json:"leveled_up"`
            NewLevel  int  `json:"new_level"`
            NewXP     int  `json:"new_xp"`
        } `json:"result"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &XPAwardResponse{
        Success:   result.Success,
        LeveledUp: result.Result.LeveledUp,
        NewLevel:  result.Result.NewLevel,
        NewXP:     result.Result.NewXP,
    }, nil
}
```

## Configuration

Add to `.env.example`:

```bash
# Admin Features
ADMIN_ROLE_ID=1234567890  # Discord role ID for admins
AUDIT_LOG_PATH=./data/audit.jsonl

# Rate Limiting
ADMIN_XP_RATE_LIMIT=10  # Max XP grants per minute
```

## Implementation Checklist

- [ ] Create `internal/audit/logger.go`
- [ ] Create `internal/audit/logger_test.go`
- [ ] Create `internal/handler/admin_job.go`
- [ ] Create `internal/handler/admin_job_test.go`
  - [ ] Test successful XP award
  - [ ] Test validation (missing fields, negative amount, exceeds max)
  - [ ] Test error handling (invalid user, invalid job)
  - [ ] Test audit logging
- [ ] Create `internal/discord/cmd_admin.go`
- [ ] Add `AwardXP` method to `internal/discord/client.go`
- [ ] Update `cmd/app/main.go`:
  - [ ] Wire up audit logger
  - [ ] Register admin endpoint `/api/v1/admin/award-xp`
  - [ ] Register Discord command
- [ ] Add configuration to `.env.example`
- [ ] Integration test:
  - [ ] Award 1000 XP to test user
  - [ ] Verify XP in database
  - [ ] Verify level up if applicable
  - [ ] Verify audit log entry
- [ ] Discord command test:
  - [ ] Mock API client
  - [ ] Test admin permission check
  - [ ] Test response formatting

## Affected Files

- [NEW] `internal/audit/logger.go`
- [NEW] `internal/audit/logger_test.go`
- [NEW] `internal/handler/admin_job.go`
- [NEW] `internal/handler/admin_job_test.go`
- [NEW] `internal/discord/cmd_admin.go`
- [MODIFY] `internal/discord/client.go`
- [MODIFY] `cmd/app/main.go`
- [MODIFY] `.env.example`

## Success Criteria

- ✅ Admin can award XP via Discord command
- ✅ Non-admins are rejected with clear error message
- ✅ XP awards trigger level-up events
- ✅ All admin actions logged to audit file
- ✅ Rate limiting prevents abuse
- ✅ Comprehensive test coverage (>85%)
- ✅ Audit log is human-readable JSON
- ✅ Command validation prevents negative/excessive amounts

## Example Usage

### Discord Command
```
/admin-award-xp @Alice job:explorer amount:1000 reason:"Bug compensation - lost search progress"
```

**Response:**
```
✅ Awarded 1000 XP to @Alice's explorer job
🎉 @Alice leveled up to level 5!
📝 Reason: Bug compensation - lost search progress
```

### Audit Log Entry
```json
{
  "timestamp": "2026-01-03T02:30:00Z",
  "action": "admin_award_xp",
  "admin_id": "discord-bot",
  "target_id": "user-uuid-123",
  "details": {
    "job_key": "explorer",
    "amount": 1000,
    "reason": "Bug compensation - lost search progress",
    "leveled_up": true,
    "new_level": 5
  }
}
```

## Security Considerations

1. **Permission Check:** Only users with `ADMIN_ROLE_ID` can execute command
2. **Rate Limiting:** Max 10 XP grants per minute (prevents spam)
3. **Audit Trail:** All actions logged with admin ID, target, and details
4. **Amount Validation:** Max 10,000 XP per grant (prevents accidents)
5. **API Authentication:** Consider adding API key for endpoint protection

## Future Enhancements (Not in this issue)

- Bulk XP awards (CSV upload)
- XP removal/adjustment command
- History view of admin actions
- Webhook notifications for large XP grants

## Related Issues

- [resilient-event-publishing.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/resilient-event-publishing.md) - Ensures XP awards trigger events reliably
- Implementation plan: [implementation_plan.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/implementation_plan.md)

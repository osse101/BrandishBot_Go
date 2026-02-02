# Discord Module Refactoring Patterns & Guide

**Document Version:** 1.0
**Last Updated:** 2026-01-14
**Context:** BrandishBot Go - Discord command handling refactoring

## Overview

This guide documents patterns, strategies, and learnings from the Discord module refactoring exercise. Use this as a reference when performing similar refactoring tasks in other parts of the codebase.

## Key Principle

**Replace N-line duplicate patterns with 1-line helper calls** by identifying:
1. The pattern (structure that repeats)
2. The variation (what changes between instances)
3. The abstraction (function that encapsulates the pattern)

---

## Pattern Identification Checklist

### ✅ How to Find Duplications

| Method | Command | Example |
|--------|---------|---------|
| **Grep for pattern** | `grep -n "pattern"` | `grep -n "InteractionResponseEdit"` |
| **Count across files** | `grep -c "pattern" file.go` | Find top duplicators |
| **Identify variations** | Manual inspection | See what changes between instances |
| **Calculate savings** | `lines × occurrences` | 8 lines × 30 instances = 240 lines |

### ✅ When to Refactor

**Good candidate for refactoring:**
- Pattern repeats 3+ times
- Code block is 5+ lines
- Same logic, different data
- Errors prone if not consistent

**Not worth refactoring:**
- Only appears 1-2 times
- Will likely diverge in future
- Already uses helpers
- Premature abstraction

---

## Refactoring Levels (Low to High Effort)

### Level 1: Direct Replacement (10-15 minutes per file)

**Situation:** Helper already exists but not being used
**Action:** Replace inline code with existing helper
**Example:** Using `i.Member.User` inline instead of `getInteractionUser(i)`

```go
// BEFORE - inline (2 instances across 4 files)
user := i.Member.User
if user == nil {
    user = i.User
}

// AFTER - using existing helper (1 call)
user := getInteractionUser(i)
```

**Impact:** 12 lines saved per file × 4 files = 48 lines total

**Effort:** ⭐ (Grep, replace, done)

---

### Level 2: Create Single-Purpose Helper (30-45 minutes)

**Situation:** Same 5-8 line pattern repeats with minor variations
**Action:** Extract into new helper function
**Example:** Embed creation + sending pattern

```go
// BEFORE - repeated 30+ times
embed := &discordgo.MessageEmbed{
    Title:       title,
    Description: msg,
    Color:       color,
    Footer: &discordgo.MessageEmbedFooter{
        Text: "BrandishBot",
    },
}
if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
    Embeds: &[]*discordgo.MessageEmbed{embed},
}); err != nil {
    slog.Error("Failed to send response", "error", err)
}

// AFTER - two helpers replace all instances
embed := createEmbed(title, msg, color, "")
sendEmbed(s, i, embed)
```

**Helper Structure:**
```go
// New helper - encapsulates the pattern
func sendEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
    if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
        Embeds: &[]*discordgo.MessageEmbed{embed},
    }); err != nil {
        slog.Error("Failed to send response", "error", err)
    }
}
```

**Impact:** 8 lines × 30 instances = 240 lines saved

**Effort:** ⭐⭐ (Define helper, test it, replace instances)

---

### Level 3: Create Helper with Configuration (45-90 minutes)

**Situation:** Pattern has multiple variations that need flexibility
**Action:** Create parameterized helper
**Example:** Error registration helper that supports friendly vs. generic errors

```go
// BEFORE - appears 18 times with variations
_, err := client.RegisterUser(user.Username, user.ID)
if err != nil {
    slog.Error("Failed to register user", "error", err)
    respondError(s, i, "Error connecting to game server.")  // OR respondFriendlyError
    return
}

// AFTER - one helper with boolean flag
if !ensureUserRegistered(s, i, client, user, friendlyError) {
    return
}
```

**Helper Design:**
```go
// Handles both error cases with one boolean
func ensureUserRegistered(s *discordgo.Session, i *discordgo.InteractionCreate,
    client *APIClient, user *discordgo.User, friendlyError bool) bool {
    _, err := client.RegisterUser(user.Username, user.ID)
    if err != nil {
        if friendlyError {
            respondFriendlyError(s, i, err.Error())
        } else {
            respondError(s, i, "Error connecting to game server.")
        }
        return false
    }
    return true
}
```

**Parameters:** Boolean or enum flags for behavior variations
**Returns:** Success/failure to enable caller to early-return
**Impact:** 6 lines × 18 instances = 108 lines saved

**Effort:** ⭐⭐⭐ (Design flexibility, test both branches, verify usage)

---

### Level 4: Create Constants for Magic Strings (10 minutes)

**Situation:** Same string appears 20+ times
**Action:** Extract to named constant
**Example:** Footer text constants

```go
// BEFORE - magic strings
Footer: &discordgo.MessageEmbedFooter{
    Text: "BrandishBot",  // Appears 20+ times
}
Footer: &discordgo.MessageEmbedFooter{
    Text: "BrandishBot Admin",  // Appears 5+ times
}

// AFTER - use constants
const (
    FooterBrandishBot      = "BrandishBot"
    FooterBrandishBotAdmin = "BrandishBot Admin"
)

// Usage: createEmbed(title, msg, color, FooterBrandishBotAdmin)
```

**Benefits:**
- Single point to update footer text
- IDE autocomplete for valid options
- Prevents typos
- Documents intent

**Impact:** 25 instances standardized, 1 place to change

**Effort:** ⭐ (Define constants, find-replace)

---

## Implementation Workflow

### Phase 1: Analysis (5-10 minutes)

```bash
# Find all occurrences of the pattern
grep -rn "Pattern.*Name" internal/discord/

# Count which files have the most
grep -c "Pattern" internal/discord/cmd_*.go | sort -t: -k2 -rn

# Estimate savings
lines=8
occurrences=30
echo "$((lines * occurrences))"  # Output: 240 lines
```

### Phase 2: Design (10-15 minutes)

1. **Identify the pattern** - What repeats?
2. **Identify variations** - What changes?
3. **Design the helper** - What parameters needed?
4. **Test locally** - Does it compile?

Example decision tree:
```
Pattern found 20+ times?
├─ YES: Continue to step 2
└─ NO: Don't refactor (premature)

Multiple variations (3+)?
├─ YES: Use boolean/enum parameter
└─ NO: Simple helper, maybe inline logic

Will code diverge in future?
├─ YES: More flexible design (harder but worth it)
└─ NO: Simple extraction is fine
```

### Phase 3: Implementation (30-120 minutes based on level)

1. **Create helper** in appropriate location (usually commands.go for Discord)
2. **Add documentation** with examples
3. **Test** - does it compile, does behavior match?
4. **Replace instances** - use find/replace with care
5. **Verify build** - run `go build ./cmd/...`

### Phase 4: Validation (5-10 minutes)

```bash
# Verify all instances replaced
grep -c "old_pattern" internal/discord/cmd_*.go  # Should be 0

# Verify new helper usage
grep -c "new_helper" internal/discord/cmd_*.go   # Should be N

# Build to verify no compilation errors
go build ./cmd/discord
```

---

## Common Patterns in Discord Commands

### Pattern 1: Deferred Response (4 lines → 1 line)

**Pattern:** Long-running operations need deferred response

```go
// BEFORE (4 lines)
if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
    Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
}); err != nil {
    slog.Error("Failed to send deferred response", "error", err)
    return
}

// AFTER (1 line)
if !deferResponse(s, i) { return }

// In helper location (commands.go)
func deferResponse(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
    if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
    }); err != nil {
        slog.Error("Failed to send deferred response", "error", err)
        return false
    }
    return true
}
```

**Occurrences:** 15-20 per module
**Savings:** ~60 lines
**Difficulty:** ⭐ (Straightforward extraction)

---

### Pattern 2: Embed Send (8 lines → 1 line)

**Pattern:** Creating and sending embeds

```go
// BEFORE (8 lines)
embed := &discordgo.MessageEmbed{
    Title:       "Title",
    Description: msg,
    Color:       0x3498db,
    Footer: &discordgo.MessageEmbedFooter{
        Text: "BrandishBot",
    },
}
if _, err := s.InteractionResponseEdit(...) { ... }

// AFTER (2 lines)
embed := createEmbed("Title", msg, 0x3498db, "")
sendEmbed(s, i, embed)
```

**Occurrences:** 25-35 per module
**Savings:** ~150 lines
**Difficulty:** ⭐⭐ (Need two helpers)

---

### Pattern 3: User Extraction (3 lines → 1 line)

**Pattern:** Get user from member or direct message

```go
// BEFORE (3 lines)
user := i.Member.User
if user == nil {
    user = i.User
}

// AFTER (1 line)
user := getInteractionUser(i)
```

**Occurrences:** 4-6 per module
**Savings:** ~12 lines
**Difficulty:** ⭐ (Simple wrapper)

---

### Pattern 4: User Registration (6 lines → 1 line)

**Pattern:** Register user with error handling

```go
// BEFORE (6 lines)
_, err := client.RegisterUser(user.Username, user.ID)
if err != nil {
    slog.Error("Failed to register user", "error", err)
    respondError(s, i, "Error connecting...")
    return
}

// AFTER (1 line)
if !ensureUserRegistered(s, i, client, user, false) { return }
```

**Occurrences:** 15-20 per module
**Savings:** ~90 lines
**Difficulty:** ⭐⭐ (Two variants: friendly vs. generic error)

---

## Refactoring Checklist

### Before Starting
- [ ] Run current tests - establish baseline
- [ ] Count duplications with grep
- [ ] Estimate lines to be saved
- [ ] Decide which patterns to tackle (prioritize high-impact)

### During Implementation
- [ ] Create helpers with clear names
- [ ] Add documentation with usage examples
- [ ] Test each helper independently
- [ ] Use find-replace carefully (review each match)
- [ ] Build frequently to catch errors early

### After Completion
- [ ] Verify zero old patterns remain (grep should find 0)
- [ ] Build passes without errors
- [ ] Run tests (should be identical behavior)
- [ ] Commit changes with clear message
- [ ] Document learnings for next time

---

## Decision Matrix: Should I Refactor?

```
             Low Effort          Medium Effort        High Effort
Low Impact   ✅ Nice to have     ❌ Skip              ❌ Skip
             (10-20 lines)

Medium       ✅ Definitely       ✅ Probably worth    ❌ Depends on
Impact       (50-100 lines)      (100-200 lines)     future changes

High Impact  ✅✅ Always         ✅✅ Always          ✅ Usually worth
             (200+ lines)        (200+ lines)        (200+ lines)
```

**Example Decision:**
- Pattern appears 30 times (High Impact)
- Saves 240 lines (High Impact)
- Takes 45 minutes to implement (Medium Effort)
- **Decision:** ✅ Definitely refactor

---

## Metrics & Measurement

### How to Measure Success

1. **Lines Removed:**
   - Formula: `pattern_lines × instances_before - helper_lines`
   - Example: `8 × 30 - 15 = 225 lines saved`

2. **Code Duplication Reduction:**
   - Before: Count unique code blocks
   - After: Count helper function usage
   - Improvement: `(before - after) / before × 100%`

3. **Maintainability:**
   - Single point of change for business logic
   - All instances use same error handling
   - Easier to add new similar features

4. **Build Time:**
   - Should be identical or faster
   - Helpers are inlined by compiler

---

## Discord Module Helpers Summary

| Helper | Lines Reduced | Instances | Difficulty | Status |
|--------|--------------|-----------|-----------|--------|
| `deferResponse()` | 60 | 18 | ⭐ | ✅ Done |
| `getInteractionUser()` | 12 | 4 | ⭐ | ✅ Done |
| `getOptions()` | 5 | 10+ | ⭐ | ✅ Done |
| `sendEmbed()` | 150+ | 23 | ⭐⭐ | ✅ Done |
| `createEmbed()` | 50 | 30+ | ⭐⭐ | ✅ Done |
| `ensureUserRegistered()` | 90 | 8 | ⭐⭐ | ✅ Done |
| Footer constants | 0 (improve) | 25 | ⭐ | ✅ Done |
| **TOTAL** | **~400 lines** | **120+** | **Avg ⭐⭐** | **✅ Done** |

---

## Lessons Learned

### ✅ What Went Well

1. **Systematic approach** - Grep to find, count to prioritize, implement in phases
2. **Helpers with parameters** - Boolean flags provide flexibility without explosion
3. **Documentation** - Comments on helpers prevent future duplicate code
4. **Build verification** - Caught issues immediately with `go build`
5. **Low-to-high effort** - Doing simple replacements first built momentum

### ⚠️ Challenges & Solutions

| Challenge | Solution |
|-----------|----------|
| **Too many patterns** | Prioritize by impact (occurrences × lines saved) |
| **Unsure if pattern is real** | Count it - if 3+ instances, probably yes |
| **Variations in pattern** | Use boolean/enum parameters instead of multiple helpers |
| **Breaking existing calls** | Always check usages before changing helper signature |
| **Hard to test impact** | Use grep before/after to count instances |

### 🎯 Best Practices

1. **Name helpers clearly** - `sendEmbed()` not `send()`
2. **Document with examples** - Add usage comments showing before/after
3. **Return booleans for control flow** - Enables `if !helper() { return }`
4. **Use constants for strings** - Footer text is prime candidate
5. **Batch similar changes** - Don't mix unrelated refactorings
6. **Verify with grep** - Don't rely on IDE "Find all" alone
7. **Build after each phase** - Catch errors early

---

## Applying This to Other Modules

### Checklist for Similar Refactoring

- [ ] Identify high-duplication module (look for similar cmd_*.go files)
- [ ] Grep for patterns that appear 3+ times
- [ ] Calculate potential savings (estimate lines × occurrences)
- [ ] Design helpers using "pattern → variations → abstraction"
- [ ] Implement in phases (low effort first, high effort later)
- [ ] Test build after each phase
- [ ] Document patterns for future reference
- [ ] Measure impact (lines removed, duplication %)

### Modules Likely to Benefit

- **Discord handlers** - ✅ Already done
- **API endpoints** - Similar request/response patterns
- **Database queries** - Repeated error handling
- **Admin commands** - Repetitive validation
- **Test helpers** - Common setup/teardown

---

## Quick Reference: Helper Template

```go
// HelperName does [what it does].
// [Key benefit/why you should use it]
//
// Parameters:
//   param1: [description]
//   param2: [description]
//
// Returns: [what it returns]
//
// Usage:
//   // Example 1
//   example code
//
//   // Example 2
//   example code
func HelperName(param1 Type1, param2 Type2) ReturnType {
    // Implementation
    return result
}
```

This template ensures helpers are well-documented and easy to use.

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-14 | Initial guide from Discord refactoring |
| TBD | TBD | Additional patterns as discovered |

---

## Related Documents

- `docs/ARCHITECTURE.md` - Overall system design
- `docs/development/journal.md` - Development patterns
- `internal/discord/commands.go` - Implementation of helpers

---

**End of Guide**

*For questions or additions to this guide, refer to the refactoring that generated these patterns: Discord module helper extraction (Jan 2026).*

# Info Command - Simplified UX Implementation

## Summary

Implemented **smart single-argument info lookup** for Streamerbot integration. Users can now type `!info harvest` instead of needing to know `!info farming harvest`.

## Changes Made

### Backend (Go)

1. **`internal/info/loader.go`**
   - Added `SearchTopic(topicName) (*InfoTopic, string, bool)` method
   - Searches all features for a topic by name
   - Returns topic, feature name, and found status

2. **`internal/handler/info.go`**
   - Updated feature lookup to fallback to topic search
   - When feature not found, automatically searches all topics
   - Returns appropriate error only if neither feature nor topic found

### C# Client

1. **`BrandishBotClient.Core.cs`**
   - `GetInfo(platform, feature, topic)` - supports both hierarchical and flat lookups

2. **`Models.cs`**
   - Added `InfoResponse` class with Platform/Feature/Topic/Description/Link fields

3. **`ResponseFormatter.cs`**
   - Added `FormatInfo(InfoResponse)` - returns description field

4. **`BrandishBotWrapper.cs`**
   - Simplified `GetInfo()` to accept single argument (input0)
   - Auto-detects platform from `userType` context
   - Backend determines if argument is feature or topic

## Usage

### Streamlined Commands

```
!info              → Overview
!info farming      → Farming feature
!info harvest      → Harvest topic (found under farming automatically)
!info crafting     → Crafting feature
!info compost      → Compost topic (found under farming automatically)
```

### API Behavior

**Request**: `GET /api/v1/info?platform=twitch&feature=harvest`

**Backend Logic**:

1. Search features for "harvest" → Not found
2. Search all topics for "harvest" → Found in `farming.yaml`
3. Return farming/harvest topic description

**Response**:

```json
{
  "platform": "twitch",
  "feature": "farming",
  "topic": "harvest",
  "description": "Harvest: Use !harvest to collect...",
  "link": ""
}
```

## Testing

Created `internal/info/search_test.go` - **All tests pass** ✅

```
=== RUN   TestSearchTopic/Find_harvest_topic_under_farming
=== RUN   TestSearchTopic/Find_compost_topic_under_farming
=== RUN   TestSearchTopic/Topic_does_not_exist
--- PASS: TestSearchTopic (0.00s)
```

## Benefits

✅ **Better UX**: Users don't need to know feature hierarchy
✅ **Backward Compatible**: Existing `?feature=farming&topic=harvest` still works
✅ **No Duplication**: Topic names unique across all features
✅ **Single Source**: All platforms use same unified YAML data
✅ **Automatic**: No configuration needed - just works

## Notes

- Topic names must be unique across all features (currently: harvest, compost, farmer are under farming)
- If same topic name existed in multiple features, first match wins
- Feature names take priority over topic names in search

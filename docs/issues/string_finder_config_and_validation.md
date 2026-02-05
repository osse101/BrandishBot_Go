# Issue: StringFinder Hardcoded Rules and Production Validation

## Description

The `StringFinder` system used for chat message parsing currently relies on hardcoded patterns and lacks a robust way to validate rule changes without code modifications.

### 1. Hardcoded Rules

The default rules (e.g., "Bapanada", "gary", "shedinja" -> "OBS") are hardcoded in the Go source.

- **Impact**: Adding new keywords or changing priorities requires a full application rebuild and redeployment. This is inefficient for a live-ops environment where streamers might want to add custom triggers quickly.
- **Location**: `internal/user/string_finder.go:loadDefaultRules`.

### 2. Regression Risk from Refactoring

A recent refactoring of the regex compilation logic needs production-level validation to ensure word boundaries and case-insensitivity work as expected across all platforms (Discord/Twitch).

- **Impact**: Failure to detect items in messages breaks the bot's primary passive engagement system.
- **Verification Requirement**: Manual regression testing on a staging environment with real chat input is required.

## Proposed Solution

- Externalize `StringFinder` rules to a configuration file (e.g., `configs/string_finder_rules.json`).
- Implement a reload mechanism (similar to item aliases) to update rules without restarting the server.
- Add comprehensive test cases for complex word boundary scenarios (e.g., handles punctuation, emojis, and non-latin characters if applicable).

## Status Update (2026-01-30)

Verified that `internal/user/string_finder.go` still contains hardcoded rules in `loadDefaultRules` ("Bapanada", "gary", "shedinja"). No configuration loading logic is present. The issue persists.

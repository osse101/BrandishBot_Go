# LevelUp Journal

## Critical Learnings

- **Feedback Loop Missing**: Players have no feedback for daily engagement/consistency. Implementing "Daily Login Streak" to address this.
- **Service/Repo Boundaries**: Adding logic to `RecordUserEvent` allows capturing all activities as "login" events without needing a dedicated "Login" event, but requires careful recursion handling (excluding the streak event itself).

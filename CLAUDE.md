# BrandishBot Go - Claude Code Context

Go game engine API for live chatroom gaming. This file contains Claude-specific instructions. For general project architecture and standards, see **[AGENTS.md](AGENTS.md)**.

## ⚡ Agent/Skill/MCP Usage Policy

**CRITICAL: Use specialized agents, skills, and MCPs proactively. Don't do work manually when tools are available.**

### 🤖 Task Agents (Automatic Delegation)

**Always delegate to agents for:**

| Trigger                      | Agent                             | When to Use                                                                                             |
| ---------------------------- | --------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Code review**              | `code-reviewer` or `golang-pro`   | When user asks to "review", before commits, after writing code                                          |
| **Concurrency analysis**     | `golang-pro`                      | Code with goroutines/channels/mutexes, mentions of "race", "deadlock", or working on workers/SSE/events |
| **Security audit**           | `security-auditor`                | User asks about security, reviewing auth/API endpoints, before production                               |
| **Performance optimization** | `performance-engineer`            | User mentions "slow", "optimize", "bottleneck", database query optimization                             |
| **Database design**          | `sql-pro` or `database-architect` | Designing schemas, complex queries, optimizing indexes                                                  |
| **Codebase exploration**     | `Explore`                         | Finding patterns across files, "where is X", understanding code structure                               |
| **Constant extraction**      | `hardcoded-constants-extractor`   | After writing handlers/commands, user mentions "magic numbers"                                          |
| **Test generation**          | `test-automator`                  | Writing new features, test coverage requests, integration tests                                         |
| **Refactoring**              | `golang-pro`                      | Reducing duplication, improving code quality, modernizing patterns                                      |

**Agent Chaining (Multi-Agent Workflows):**

Chain agents automatically for complex tasks:

- **"Review this code"** → `code-reviewer` → `security-auditor` → `golang-pro`
- **"Optimize database"** → `Explore` → `sql-pro` → `performance-engineer`
- **"Add feature X"** → `Plan` → `backend-architect` → `test-automator`
- **"Find and fix Y"** → `Explore` → specialized agent → implement fix

---

### 🔌 MCPs (Data Access & Memory)

**Use MCPs for direct data operations:**

| MCP                         | When to Use                                                       | Example                                           |
| --------------------------- | ----------------------------------------------------------------- | ------------------------------------------------- |
| `mcp__postgres__query`      | Direct database queries for debugging/validation                  | `SELECT * FROM progression_nodes WHERE tier = 3`  |
| `mcp__memory__*`            | Store technical decisions, architecture patterns, lessons learned | Track refactoring decisions, document pain points |
| `mcp__memory__search_nodes` | Retrieve past decisions/patterns                                  | "What did we decide about caching strategy?"      |

---

### 🔄 Decision Tree: Which Tool to Use?

```
User asks for analysis/review/optimization?
  ├─ YES → Use Task Agent (golang-pro, code-reviewer, etc.)
  └─ NO
      ├─ User wants to run commands (migrations, tests, deploy)?
      │   └─ YES → Use Skill (goose, testing, deployment) or Workflow (AGENTS.md)
      └─ User wants data/memory lookup?
          └─ YES → Use MCP (postgres query, memory search)
```

---

## Refactoring & Code Quality

### Pattern Learning Prompts

When performing these tasks, Claude should ask to fill in the complete pattern:

**Adding a New API Endpoint:**
See **[Add-Endpoint Workflow](.agent/workflows/add-endpoint.md)** for the step-by-step process.

**Adding a New Discord Command:**
**Ask:** "I'll add this command. Please confirm:

1. Command name and description?
2. Options/parameters?
3. Autocomplete needed for which fields?
4. Which API endpoint does it call?
5. Response formatting?"

---

### Development Artifacts Location

During development, create temporary artifacts in designated locations:

- **Plan documents:** `.claude/plans/*.md` (gitignored)
- **Temporary notes:** `.claude/notes/*.md` (gitignored)
- **Debug outputs:** `.claude/debug/*.log` or `.claude/debug/*.json` (gitignored)

**Never commit:** Plan documents, debug logs, temporary notes, or ephemeral test data. These locations are gitignored for this reason.

### Feature Completion Cleanup

When completing a feature or task, clean up all temporary artifacts:

- [ ] Delete plan documents (e.g., `.claude/plans/*.md`)
- [ ] Remove any temporary debugging files
- [ ] Delete test data files not needed for CI
- [ ] Clean up commented-out code blocks
- [ ] Remove TODO comments that were addressed

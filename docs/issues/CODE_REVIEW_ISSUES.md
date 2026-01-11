# Implementation Issues - Code Review Follow-up

**Created:** 2026-01-03  
**Source:** Code review of changes since 2026-01-02

These issues track the implementation of code review recommendations.

## Active Issues

### High Priority

1. **[refactor-username-methods.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/refactor-username-methods.md)**
   - **Effort:** 3-4 hours | **Complexity:** 7/10
   - Eliminate ~500 lines of code duplication in username-based inventory methods
   - Extract internal helpers for transaction logic

2. **[resilient-event-publishing.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/resilient-event-publishing.md)**
   - **Effort:** 4-6 hours | **Complexity:** 8/10
   - Implement fire-and-forget with retry queue
   - Ensure XP awards never fail due to event errors
   - Dead-letter logging for permanent failures

3. **[document-event-system.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/document-event-system.md)**
   - **Effort:** 1-2 hours | **Complexity:** 3/10
   - Create architecture overview with Mermaid diagrams
   - Event catalog with payload schemas
   - Developer guide for adding new events

4. **[admin-xp-award-command.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/admin-xp-award-command.md)**
   - **Effort:** 2-3 hours | **Complexity:** 4/10
   - Discord command: `/admin-award-xp @user job:explorer amount:1000`
   - Audit logging and rate limiting
   - Use cases: Bug compensation, events, testing

### Medium Priority

5. **[api-versioning-framework.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/api-versioning-framework.md)**
   - **Effort:** 4-5 hours | **Complexity:** 6/10
   - URL-based versioning: `/api/v1/`, `/api/v2/`
   - Immediate migration of all endpoints to v1
   - Client version tracking via headers

## Total Effort

**14-20 hours** across 5 issues

## Workflow

When an issue is completed:
1. Add `RESOLVED` at the top of the file
2. Move to `docs/issues/resolved/` directory
3. Update this index

Example:
```bash
# Mark as resolved
echo -e "RESOLVED\n\n$(cat refactor-username-methods.md)" > refactor-username-methods.md

# Move to resolved
mv refactor-username-methods.md resolved/
```

## Related Documentation

- Code Review: [code_review.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/code_review.md)
- Implementation Plan: [implementation_plan.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/implementation_plan.md)
- Summary: [summary.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/summary.md)

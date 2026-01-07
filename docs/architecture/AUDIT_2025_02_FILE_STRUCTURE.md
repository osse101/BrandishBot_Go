# Audit Report: File Structure and Architecture (Feb 2025)

## Overview
This document serves as an audit of the current file structure, package organization, and mock usage within the `BrandishBot_Go` project. The goal is to identify technical debt, architectural inconsistencies, and opportunities for improvement.

## 1. Project Layout & Package Structure

### Current State
The project utilizes a **Hybrid Structure**:
- **Package-by-Feature**: Core domains like `internal/user` and `internal/crafting` encapsulate their business logic, models, and local helpers.
- **Package-by-Layer**: Technical layers like `internal/handler` (HTTP transport) and `internal/database/postgres` (Data access) are separated into their own packages.

### Analysis
- **Pros**:
    - Separation of concerns: HTTP handlers are distinct from business logic.
    - Swappable implementations: Database logic is isolated in `postgres`, making it easier to swap or mock.
- **Cons**:
    - **Discovery**: To understand the full flow of a "User" feature, one must navigate between `internal/handler/user.go`, `internal/user/service.go`, and `internal/database/postgres/user.go`.
    - **Ambiguity**: Some files in feature packages have ambiguous names (see Navigability).

## 2. Mock Repository Inconsistency (Major Finding)

### Findings
There is a significant split in how mocks are implemented and utilized:

1.  **Manual Mocks (`internal/user/mock_repository.go`)**:
    -   **Status**: Actively used in tests (e.g., `internal/user/service_test.go`).
    -   **Issues**: Brittle. Requires manual updates whenever the `Repository` interface changes.
    -   **Debt**: Explicitly marked with `// TODO: Replace this with generated mocks eventually.`

2.  **Generated Mocks (`mocks/mock_user_repository.go`)**:
    -   **Status**: Exists and appears up-to-date (generated via `mockery`).
    -   **Usage**: **Unused** in the core service tests checked during this audit.
    -   **Benefit**: Automatically synced with interfaces; uses standard `stretchr/testify/mock`.

### Impact
The maintenance burden is doubled. Developers must update the manual mock for every schema change, ignoring the tool-generated mock intended to solve this exact problem.

## 3. Navigability & Naming

### Observations
-   **`internal/user/item_handlers.go`**:
    -   **Content**: Contains business logic for processing item actions (e.g., Opening Lootboxes, using Blasters).
    -   **Naming**: The suffix `_handlers` is confusing as it conflicts with the HTTP "Handler" terminology used in `internal/handler`.
    -   **Recommendation**: Rename to `item_actions.go` or `item_processor.go` to clearly denote *domain logic*.

-   **`internal/user/string_finder.go`**:
    -   **Content**: text search utilities.
    -   **Placement**: Currently in `user` domain.
    -   **Recommendation**: Move to `internal/utils` or a dedicated `internal/search` if logic expands.

## 4. Recommendations

1.  **Standardize Mocks**:
    -   **Action**: Refactor `internal/user/service_test.go` to use `mocks/mock_user_repository.go`.
    -   **Cleanup**: Delete `internal/user/mock_repository.go` once migration is complete.

2.  **Rename Ambiguous Files**:
    -   Rename `internal/user/item_handlers.go` -> `internal/user/item_actions.go`.

3.  **Documentation**:
    -   Update `docs/development/FEATURE_DEVELOPMENT_GUIDE.md` to explicitly enforce the use of `make mocks` and generated mocks for new features.

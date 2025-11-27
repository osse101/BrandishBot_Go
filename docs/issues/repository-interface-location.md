# Repository Interface Location Refactoring

## Issue  
Repository interfaces are currently defined in service packages (e.g., `internal/user/service.go`), creating coupling between the service layer and the persistence interface definition.

## Current Structure
```
internal/user/service.go - Contains Repository interface
internal/database/postgres/user.go - Implements interface
```

This violates clear separation of concerns as the service package needs to define what the repository should provide.

## Proposed Solution

### Option 1: Dedicated Repository Package (Recommended)
Create a new `internal/repository/` package:

```
internal/repository/
├── user.go        # type UserRepository interface
├── stats.go       # type StatsRepository interface  
├── progression.go # type ProgressionRepository interface
└── economy.go     # type EconomyRepository interface
```

**Benefits:**
- Clear separation of concerns
- Single location for all repository interfaces
- Easier to understand system architecture
- Follows dependency inversion principle

**Drawbacks:**
- Breaking change requiring updates across codebase
- All service packages need to import repository package
- Migration effort required

### Option 2: Document Current Pattern (Non-breaking)
Add comprehensive documentation to existing interfaces:

```go
// internal/user/service.go

// Repository defines the interface for user persistence.
// 
// Implementation location: internal/database/postgres/user.go
// 
// This interface is defined here (in the service package) to follow the
// dependency inversion principle - the service layer defines what it needs,
// and the database layer implements it.
//
// Testing: Use mock implementations in *_test.go files
type Repository interface {
    // ...
}
```

**Benefits:**
- No breaking changes
- Current code continues working
- Clear documentation of pattern

**Drawbacks:**
- Doesn't fully resolve coupling
- Pattern may confuse new contributors
- Less discoverable than dedicated package

## Impact Analysis

### Files Affected (Option 1):
- `internal/user/service.go` - Move Repository interface
- `internal/stats/service.go` - Move StatsRepository interface
- `internal/progression/service.go` - Move ProgressionRepository interface
- `internal/economy/service.go` - Move EconomyRepository interface  
- `internal/crafting/service.go` - Move interfaces
- All service test files - Update imports
- All database implementation files - Update interface reference
- `cmd/app/main.go` - May need import updates

### Estimated Effort
- **Planning**: 1 hour
- **Implementation**: 4-6 hours
- **Testing**: 2-3 hours
- **Total**: ~1 day

## Recommendation

**Phase 1** (Immediate): Implement Option 2
- Add comprehensive documentation to existing interfaces
- Create architectural decision record (ADR)
- Include in onboarding documentation

**Phase 2** (Future): Implement Option 1
- Schedule during refactoring sprint
- Create migration guide
- Update all affected code
- Comprehensive testing

## Priority
**Low-Medium** - Architectural improvement that would benefit long-term maintainability but is not blocking current development.

## References
- Feature Development Guide: Line 143 Repository Layer section
- Code Quality Recommendations: Item #6
- Dependency Inversion Principle (SOLID)

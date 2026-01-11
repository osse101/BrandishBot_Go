# Agent Tools & Techniques Journal

> Documenting learnings, patterns, and best practices from AI-assisted development

---

## 2025-12-23: Test Suite Fixes & Mock Management

### Context
Fixing broken tests after refactoring cooldown service and adding new interface methods.

### What Worked ✅

#### 1. **Systematic Diagnosis**
- Created comprehensive test diagnosis document
- Categorized failures by root cause
- Prioritized fixes by impact

**Lesson**: Always diagnose before fixing. Understanding the full scope prevents thrashing.

#### 2. **Interface-First Verification**
Before fixing mocks, checked actual interface signatures:
```bash
grep "AddContribution" internal/progression/service.go
```

**Lesson**: Verify source of truth before making changes to mocks.

#### 3. **Incremental Testing**
After each fix, ran relevant test to verify:
```bash
go test ./internal/handler -v -run TestHandleAddItem
```

**Lesson**: Fast feedback loops catch issues early.

### What Didn't Work ❌

#### 1. **Using `sed` for Multi-line Go Code Edits**

**Problems Encountered**:
- Tabs converted to spaces causing syntax errors
- Line number shifts after deletions broke subsequent commands
- Multi-line replacements created duplicate functions
- Heredoc content had inconsistent whitespace

**Example Failure**:
```bash
# This created 3 duplicate functions!
sed -i '914,918d' file.go
sed -i '913r /tmp/fix.txt' file.go
```

**Root Cause**: Sequential sed operations don't account for line number changes.

**Better Approach**: Use `replace_file_content` or `multi_replace_file_content` tools, or manual editing.

#### 2. **Mockery Configuration Complexity**

**Problem**: First attempt at mockery config used unsupported options, generated files in wrong locations.

**What Happened**:
```yaml
# This failed:
output: "{{.InterfaceDir}}/mocks"
```

**Lesson**: For simpler cases, use command-line mockery per-interface rather than wrestling with config files.

---

## Key Patterns & Best Practices

### Mock Testing

#### Pattern: Transaction Mock Setup
```go
// GOOD: Explicit transaction flow
mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
mockTx.On("GetInventory", ctx, userID).Return(inventory, nil)
mockTx.On("UpdateInventory", ctx, userID, mock.Anything).Return(nil)
mockTx.On("Commit", ctx).Return(nil)

// Also mock Rollback for error paths
mockTx.On("Rollback", ctx).Return(nil).Maybe()
```

**Lesson**: When service uses transactions, mocks must reflect the full transaction lifecycle.

#### Pattern: Interface Implementation Completeness
```go
// MockTx must implement ALL methods of repository.Tx
type MockTx struct {
    mock.Mock
}

func (m *MockTx) Commit(ctx context.Context) error
func (m *MockTx) Rollback(ctx context.Context) error  
func (m *MockTx) GetInventory(...)
func (m *MockTx) UpdateInventory(...)
```

**Lesson**: Incomplete mocks cause compilation errors. Use interface as checklist.

### Code Editing Tools

#### When to Use What

| Tool | Best For | Avoid For |
|------|----------|-----------|
| `sed` | Single-line text, config files | Go code, tabs, multi-line |
| `replace_file_content` | Single contiguous block | Multiple scattered changes |
| `multi_replace_file_content` | Multiple non-contiguous blocks | Entire file rewrites |
| Manual editing (IDE) | Complex refactors, syntax-sensitive | Bulk renames across files |

#### Example: Proper Tool Selection

**Scenario**: Fix mock method signature

**Bad**: 
```bash
sed -i 's/AddContribution(ctx context.Context, userID string, value int, source string)/AddContribution(ctx context.Context, amount int)/'
```

**Good**:
```go
// Use replace_file_content with exact target
TargetContent: `func (m *MockProgressionService) AddContribution(ctx context.Context, userID string, value int, source string) error {
	args := m.Called(ctx, userID, value, source)
	return args.Error(0)
}`

ReplacementContent: `func (m *MockProgressionService) AddContribution(ctx context.Context, amount int) error {
	args := m.Called(ctx, amount)
	return args.Error(0)
}`
```

---

## Common Pitfalls

### 1. **Mock Drift**
**Problem**: Hand-written mocks fall out of sync with interfaces.

**Solutions**:
- Auto-generate with mockery
- Add CI check: `mockery --dry-run`
- Document interface changes in PRs

### 2. **Test Dependencies**
**Problem**: Tests break when service constructor signature changes.

**Example**:
```go
// Before: NewService(repo, stats, jobs, lootbox, naming)
// After: NewService(repo, stats, jobs, lootbox, naming, cooldown, devMode)
// Result: ALL tests break
```

**Solutions**:
- Use test builders/factories
- Consider functional options pattern
- Document breaking changes

### 3. **Incomplete Transaction Mocks**
**Problem**: Forgot to mock `Commit()` or `BeginTx()`.

**Symptom**:
```
panic: mock: I don't know what to return because the method call was unexpected.
        BeginTx(context.backgroundCtx)
```

**Solution**: Always mock complete transaction flow when service uses `BeginTx`.

---

## Automation Opportunities

### 1. **Mock Generation**
```makefile
mock-generate:
	@mockery --name=Service --dir=internal/user --output=internal/user/mocks

mock-check:
	@mockery --dry-run || echo "Mocks out of date!"
```

### 2. **Test Coverage Validation**
```makefile
test-coverage-check:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total | awk '{if ($$3 < 80) exit 1}'
```

### 3. **Interface Compatibility Check**
```bash
# Check if mock implements interface
go build -o /dev/null ./internal/handler/inventory_test.go
```

## References

- [Mockery Documentation](https://vektra.github.io/mockery/)
- [Testify Mock Guide](https://pkg.go.dev/github.com/stretchr/testify/mock)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)

---

*Last Updated: 2025-12-23*

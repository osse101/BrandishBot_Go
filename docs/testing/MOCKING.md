# Mocking with Mockery

## Overview

BrandishBot_Go uses [mockery](https://github.com/vektra/mockery) to auto-generate type-safe mocks for testing. This replaces manual mock implementations with professional, maintainable generated code.

## Quick Start

### Using Generated Mocks

```go
import "github.com/osse101/BrandishBot_Go/mocks"

func TestHandleAddItem(t *testing.T) {
    // Create mock with mockery
    mockSvc := mocks.NewMockUserService(t)
    
    // Set expectations
    mockSvc.On("AddItem", mock.Anything, "twitch", "id", "user", "Sword", 1).
        Return(nil)
    
    // Use in test
    handler := HandleAddItem(mockSvc)
    // ... test code
    
    // Verify expectations met
    mockSvc.AssertExpectations(t)
}
```

### Regenerating Mocks

When interfaces change:

```bash
make mocks
```

## Available Mocks

All mocks are in the `mocks/` package with explicit naming:

**Handler Layer:**
- `MockUserService` - User service operations
- `MockEconomyService` - Economy/shop operations
- `MockProgressionService` - Progression unlocks
- `MockCraftingService` - Crafting/upgrades
- `MockEventBus` - Event publishing

**Repository Layer:**
- `MockUserRepository` - User data access
- `MockEconomyRepository` - Economy data
- `MockRepositoryTx` - Transactions

**See [.mockery.yaml](file:///home/osse1/projects/BrandishBot_Go/.mockery.yaml) for complete list**

## Patterns

### Handler Testing (Use Mockery)

**Best for:** Simple dependency mocking

```go
func TestHandler(t *testing.T) {
    // ✅ Use mockery for clean, type-safe handler tests
    mockSvc := mocks.NewMockUserService(t)
    mockBus := mocks.NewMockEventBus(t)
    
    mockSvc.On("GetUser", "123").Return(user, nil)
    mockBus.On("Publish", mock.Anything, mock.Anything).Return(nil)
    
    handler := NewHandler(mockSvc, mockBus)
    // ... test
}
```

### Service Testing (Functional Mocks)

**Best for:** Complex state management

```go
// ✅ Keep functional in-memory mocks for service tests
type MockRepository struct {
    users map[string]*domain.User
}

func (m *MockRepository) GetUser(id string) (*domain.User, error) {
    if user, ok := m.users[id]; ok {
        return user, nil
    }
    return nil, ErrNotFound
}
```

### When to Use Which

| Test Type | Use Mockery | Use Functional Mock |
|-----------|-------------|---------------------|
| Handler/Controller | ✅ Yes | ❌ No |
| Service/Business Logic | Maybe | ✅ Preferred |
| Repository/Data Layer | ✅ Yes | Only if complex |
| Simple Utilities | ❌ No mocks needed | ❌ No mocks needed |

## Mock Expectations

### Basic Setup

```go
// Simple return value
mock.On("MethodName", arg1, arg2).Return(returnValue, nil)

// Multiple calls
mock.On("GetUser", "123").Return(user1, nil).Once()
mock.On("GetUser", "456").Return(user2, nil).Once()

// Any argument
mock.On("AddItem", mock.Anything, mock.Anything).Return(nil)

// Specific + any
mock.On("AddItem", mock.Anything, "platform", mock.Anything).Return(nil)
```

### Argument Matchers

```go
// Custom matcher
mock.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
    return evt.Type == "item.sold"
})).Return(nil)

// Type matcher
mock.On("UpdateUser", mock.AnythingOfType("*domain.User")).Return(nil)
```

### Return Behaviors

```go
// Error cases
mock.On("GetUser", "invalid").Return(nil, ErrNotFound)

// Panic
mock.On("DangerousMethod").Panic("Something went wrong")

// Run custom function
mock.On("ProcessItem", mock.Anything).Run(func(args mock.Arguments) {
    item := args.Get(0).(*domain.Item)
    item.Processed = true
}).Return(nil)
```

## Common Patterns

### Testing Error Paths

```go
func TestHandler_ErrorHandling(t *testing.T) {
    mockSvc := mocks.NewMockUserService(t)
    
    // Setup error expectation
    mockSvc.On("GetUser", "missing").
        Return(nil, user.ErrNotFound)
    
    handler := NewHandler(mockSvc)
    result, err := handler.GetUser("missing")
    
    assert.Error(t, err)
    assert.ErrorIs(t, err, user.ErrNotFound)
}
```

### Testing Event Publishing

```go
func TestHandler_PublishesEvent(t *testing.T) {
    mockBus := mocks.NewMockEventBus(t)
    
    // Expect specific event
    mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
        return evt.Type == "user.created" && evt.UserID == "123"
    })).Return(nil)
    
    // ... test that triggers event
    
    mockBus.AssertExpectations(t)
}
```

### Table-Driven with Mocks

```go
func TestUserService(t *testing.T) {
    tests := []struct {
        name      string
        userID    string
        setupMock func(*mocks.MockUserRepository)
        wantErr   bool
    }{
        {
            name:   "user exists",
            userID: "123",
            setupMock: func(m *mocks.MockUserRepository) {
                m.On("GetUserByID", mock.Anything, "123").
                    Return(&domain.User{ID: "123"}, nil)
            },
            wantErr: false,
        },
        {
            name:   "user not found",
            userID: "missing",
            setupMock: func(m *mocks.MockUserRepository) {
                m.On("GetUserByID", mock.Anything, "missing").
                    Return(nil, user.ErrNotFound)
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := mocks.NewMockUserRepository(t)
            tt.setupMock(mockRepo)
            
            svc := user.NewService(mockRepo)
            _, err := svc.GetUser(context.Background(), tt.userID)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Configuration

Mocks are configured in [.mockery.yaml](file:///home/osse1/projects/BrandishBot_Go/.mockery.yaml):

```yaml
with-expecter: true  # Enable EXPECT() pattern
dir: "mocks"         # Output directory
outpkg: "mocks"      # Package name

packages:
  github.com/osse101/BrandishBot_Go/internal/user:
    config:
      filename: "mock_user_{{.InterfaceName | snakecase}}.go"
      mockname: "MockUser{{.InterfaceName}}"
    interfaces:
      Service:
      Repository:
```

**Key settings:**
- `with-expecter: true` - Enables type-safe EXPECT() pattern
- Custom filenames per package - Prevents naming collisions
- Explicit mock names - `MockUserService`, not just `MockService`

## Adding New Mocks

1. Add interface to `.mockery.yaml`:

```yaml
packages:
  github.com/osse101/BrandishBot_Go/internal/mypackage:
    config:
      filename: "mock_mypackage_{{.InterfaceName | snakecase}}.go"
      mockname: "MockMypackage{{.InterfaceName}}"
    interfaces:
      MyNewInterface:
```

2. Regenerate:

```bash
make mocks
```

3. Use in tests:

```go
mock := mocks.NewMockMypackageMyNewInterface(t)
```

## Troubleshooting

### Mock Not Found

**Error:** `undefined: mocks.NewMockXXX`

**Solution:** Interface not in `.mockery.yaml` or mocks not generated

```bash
make mocks  # Regenerate all mocks
```

### Method Not Available

**Error:** `mock.EXPECT().MethodName undefined`

**Solution:** Interface changed but mocks not updated

```bash
make mocks  # Regenerate after interface changes
```

### Expectation Not Met

**Error:** `FAIL: 0 out of 1 expectation(s) were met`

**Solution:** Mock expected a call that didn't happen or arguments didn't match

```go
// Debug by checking exact arguments
mockSvc.On("Method", mock.Anything).Return(nil)  // Less specific
// vs
mockSvc.On("Method", "exact", "args").Return(nil)  // More specific
```

## Best Practices

✅ **DO:**
- Use mockery for handler/controller tests
- Regenerate mocks when interfaces change
- Use `mock.Anything` for irrelevant arguments
- Verify expectations with `AssertExpectations(t)`

❌ **DON'T:**
- Mock everything - keep functional mocks for complex state
- Forget to call `AssertExpectations(t)`
- Over-specify expectations (brittle tests)
- Create manual mocks for new code

## Package Structure for Mocks

Recommended structure for packages with mocks:

```
internal/<package>/
├── repository.go           # Interface definition (or wrapper)
├── fake_repository.go      # Optional: Stateful fake (manual)
├── mocks/
│   └── mock_repository.go  # Generated by mockery
└── *_test.go               # Tests using either mock type
```

## Advanced: Using Stateful Fakes

While Mockery handles most cases, stateful fakes are better for integration-style tests where you need to verify complex state transitions without setting up dozens of expectations.

**Example: Integration Test with State Manipulation**

```go
package user_test

import (
    "testing"
    "github.com/osse101/BrandishBot_Go/internal/user"
)

func TestService_ComplexWorkflow(t *testing.T) {
    // 1. Create fake with initial state (not a generated mock)
    fake := user.NewFakeRepository()
    fake.Users["alice"] = &domain.User{
        Username: "alice",
        Money: 1000,
    }

    // 2. Create service with fake
    svc := user.NewService(fake)

    // 3. Run complex workflow
    err := svc.BuyItem(context.Background(), "alice", "sword")
    require.NoError(t, err)

    // 4. Verify state changes directly
    user := fake.Users["alice"]
    assert.Equal(t, 900, user.Money)  // Spent 100
    assert.Contains(t, fake.Inventories["alice"].Slots, "sword")
}
```

## References

- [Mockery Documentation](https://vektra.github.io/mockery/)
- [Testify Mock Package](https://pkg.go.dev/github.com/stretchr/testify/mock)
- [Project Configuration](file:///home/osse1/projects/BrandishBot_Go/.mockery.yaml)
- [Handler Test Examples](file:///home/osse1/projects/BrandishBot_Go/internal/handler/inventory_test.go)

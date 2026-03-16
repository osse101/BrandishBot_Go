# Implement Persistent Shield Storage

Currently, the `ApplyShield` function in `internal/user/shield.go` only has a placeholder implementation:

```go
// ApplyShield activates shield protection for a user (blocks next weapon attacks)
// Note: Shield count is stored in-memory and will be lost on server restart
func (s *service) ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error {
    // ...
	// TODO: Implement persistent shield storage
    // ...
}
```

This needs to be implemented to properly store shields persistently and integrate with the weapon handler.

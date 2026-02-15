package user

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ApplyShield activates shield protection for a user (blocks next weapon attacks)
// Note: Shield count is stored in-memory and will be lost on server restart
func (s *service) ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error {
	log := logger.FromContext(ctx)
	log.Info("ApplyShield called", "userID", user.ID, "quantity", quantity, "is_mirror", isMirror)

	// For now, shields are stored in user metadata or a simple map
	// This is a placeholder implementation - full implementation would need persistent storage
	// The shield check would be integrated into the weapon handler

	// TODO: Implement persistent shield storage
	// For now, just log and return success
	shieldType := "standard"
	if isMirror {
		shieldType = "mirror"
	}
	log.Info("Shield applied (placeholder implementation)", "userID", user.ID, "quantity", quantity, "type", shieldType)
	return nil
}

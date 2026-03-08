package utils

import "github.com/osse101/BrandishBot_Go/internal/domain"

const InventoryLookupLinearScanThreshold = 50

type SlotKey struct {
	ItemID       int
	QualityLevel domain.QualityLevel
}

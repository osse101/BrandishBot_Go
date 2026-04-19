package itemhandler

import (
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ============================================================================
// Error Messages
// ============================================================================

// ============================================================================
// Log Messages
// ============================================================================

const (
	LogMsgHandleWeaponCalled      = "handleWeapon called"
	LogMsgHandleReviveCalled      = "handleRevive called"
	LogMsgHandleShieldCalled      = "handleShield called"
	LogMsgHandleRareCandyCalled   = "handleRareCandy called"
	LogMsgHandleTrapCalled        = "handleTrap called"
	LogMsgResourceGeneratorCalled = "ResourceGeneratorHandler called"
	LogMsgUtilityCalled           = "UtilityHandler called"

	LogMsgWeaponUsed    = "weapon used"
	LogMsgReviveUsed    = "revive used"
	LogMsgShieldApplied = "shield applied"
	LogMsgRareCandyUsed = "rare candy used"

	LogWarnWeaponNotInInventory         = "weapon not in inventory"
	LogWarnNotEnoughWeapons             = "not enough weapons in inventory"
	LogWarnTargetUsernameMissingWeapon  = "target username missing for weapon"
	LogWarnReviveNotInInventory         = "revive not in inventory"
	LogWarnNotEnoughRevives             = "not enough revives in inventory"
	LogWarnTargetUsernameMissingRevive  = "target username missing for revive"
	LogWarnShieldNotInInventory         = "shield not in inventory"
	LogWarnNotEnoughShields             = "not enough shields in inventory"
	LogWarnRareCandyNotInInventory      = "rarecandy not in inventory"
	LogWarnNotEnoughRareCandy           = "not enough rare candy in inventory"
	LogWarnJobNameMissing               = "job name missing for rare candy"
	LogWarnFailedToTimeoutUser          = "Failed to timeout user"
	LogWarnFailedToReduceTimeout        = "Failed to reduce timeout"
	LogWarnFailedToApplyShield          = "Failed to apply shield"
	LogWarnFailedToRecordLootboxJackpot = "Failed to record lootbox jackpot event"
	LogWarnFailedToRecordLootboxBigWin  = "Failed to record lootbox big-win event"
)

// ============================================================================
// User-Facing Messages
// ============================================================================

const (
	MsgLootboxEmpty    = "The lootbox was empty!"
	MsgLootboxOpened   = "Opened"
	MsgLootboxReceived = " and received: "
	MsgLootboxJackpot  = " JACKPOT! 🎰✨"
	MsgLootboxBigWin   = " BIG WIN! 💰"
	MsgLootboxNiceHaul = " Nice haul! 📦"

	MsgBlasterReasonBy = "Blasted by "
	MsgTNTReasonBy     = "Blown up by "
	MsgGrenadeReasonBy = "Blown up by "
	MsgThisReason      = "Played yourself"
	MsgShovelUsed      = " used a shovel and found "
	MsgStickUsed       = " planted a stick as a monument to their achievement!"

	LootboxDropSeparator = ", "
)

// rarecandyXPAmount defines the XP granted per rare candy.
const rarecandyXPAmount = 500

// validVideoFiltersList is the comma-separated list of valid video filters.
const validVideoFiltersList = "bloom, bw, frameskip, gameboy, glitch, matrix, outline, page, perspective, pixelate, polar, rainbow, sick, thermal, undertale, vhs, zoom"

// ============================================================================
// Weapon / Revive Timeout Tables
// ============================================================================

var weaponTimeouts = map[string]time.Duration{
	domain.ItemMissile:     60 * time.Second,
	domain.ItemHugeMissile: 6000 * time.Second,
	domain.ItemThis:        101 * time.Second,
	domain.ItemDeez:        202 * time.Second,
	domain.ItemGrenade:     60 * time.Second,
	domain.ItemTNT:         60 * time.Second,
}

var reviveRecoveryTimes = map[string]time.Duration{
	domain.ItemReviveSmall:  60 * time.Second,
	domain.ItemReviveMedium: 600 * time.Second,
	domain.ItemReviveLarge:  6000 * time.Second,
}

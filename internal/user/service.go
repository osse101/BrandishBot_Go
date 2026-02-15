package user

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// validPlatforms defines the supported platform values
var validPlatforms = map[string]bool{
	domain.PlatformTwitch:  true,
	domain.PlatformYoutube: true,
	domain.PlatformDiscord: true,
}

// timeoutInfo tracks active timeouts
type timeoutInfo struct {
	timer     *time.Timer
	expiresAt time.Time
}

// service implements the Service interface
type service struct {
	repo            repository.User
	trapRepo        repository.TrapRepository
	handlerRegistry *HandlerRegistry
	timeoutMu       sync.Mutex
	timeouts        map[string]*timeoutInfo // Keyed by "platform:username"
	lootboxService  lootbox.Service
	publisher       *event.ResilientPublisher
	statsService    stats.Service
	stringFinder    *StringFinder
	namingResolver  naming.Resolver
	cooldownService cooldown.Service
	jobService      job.Service // Job service for retrieving job levels
	eventBus        event.Bus   // Event bus for publishing timeout events
	devMode         bool        // When true, bypasses cooldowns
	userCache       *userCache  // In-memory cache for user lookups

	// Item cache: in-memory cache for item metadata (name, description, value, etc.)
	// Purpose: Reduce database queries for frequently accessed item data
	// Thread-safety: Protected by itemCacheMu (RWMutex)
	// Invalidation: Cache is populated on-demand and persists for server lifetime
	//               Item metadata is assumed immutable - if items are modified in DB,
	//               server restart is required to refresh cache
	itemCacheByName map[string]domain.Item // Primary cache by internal name
	itemIDToName    map[int]string         // Index for ID -> name lookups
	itemCacheMu     sync.RWMutex           // Protects both maps

	activeChatterTracker *ActiveChatterTracker // Tracks users eligible for random targeting

	rnd func() float64 // For RNG - allows deterministic testing

	wg sync.WaitGroup // Track background tasks for graceful shutdown
}

// Compile-time interface checks
var _ Service = (*service)(nil)
var _ InventoryService = (*service)(nil)
var _ ManagementService = (*service)(nil)
var _ AccountLinkingService = (*service)(nil)
var _ GameplayService = (*service)(nil)

// setPlatformID sets the appropriate platform-specific ID field on a user
func setPlatformID(user *domain.User, platform, platformID string) {
	switch platform {
	case domain.PlatformTwitch:
		user.TwitchID = platformID
	case domain.PlatformYoutube:
		user.YoutubeID = platformID
	case domain.PlatformDiscord:
		user.DiscordID = platformID
	}
}

func loadCacheConfig() CacheConfig {
	config := DefaultCacheConfig()

	if val := os.Getenv(EnvUserCacheSize); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.Size = size
		}
	}

	if val := os.Getenv(EnvUserCacheTTL); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil && ttl > 0 {
			config.TTL = ttl
		}
	}

	return config
}

// NewService creates a new user service
func NewService(repo repository.User, trapRepo repository.TrapRepository, statsService stats.Service, publisher *event.ResilientPublisher, lootboxService lootbox.Service, namingResolver naming.Resolver, cooldownService cooldown.Service, jobService job.Service, eventBus event.Bus, devMode bool) Service {
	return &service{
		repo:                 repo,
		trapRepo:             trapRepo,
		handlerRegistry:      NewHandlerRegistry(),
		timeouts:             make(map[string]*timeoutInfo),
		lootboxService:       lootboxService,
		publisher:            publisher,
		statsService:         statsService,
		stringFinder:         NewStringFinder(),
		namingResolver:       namingResolver,
		cooldownService:      cooldownService,
		jobService:           jobService,
		eventBus:             eventBus,
		devMode:              devMode,
		itemCacheByName:      make(map[string]domain.Item),
		itemIDToName:         make(map[int]string),
		userCache:            newUserCache(loadCacheConfig()),
		activeChatterTracker: NewActiveChatterTracker(),
		rnd:                  utils.RandomFloat,
	}
}

func getPlatformKeysFromUser(user domain.User) map[string]string {
	keys := make(map[string]string)
	if user.TwitchID != "" {
		keys[domain.PlatformTwitch] = user.TwitchID
	}
	if user.YoutubeID != "" {
		keys[domain.PlatformYoutube] = user.YoutubeID
	}
	if user.DiscordID != "" {
		keys[domain.PlatformDiscord] = user.DiscordID
	}
	return keys
}

func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgUserServiceShuttingDown)

	// 1. Stop the chatter tracker (stops cleanup loop)
	if s.activeChatterTracker != nil {
		s.activeChatterTracker.Stop()
	}

	// 2. Wait for local async tasks (like trap triggers)
	s.wg.Wait()

	// 3. Shut down the publisher (waits for pending events)
	if s.publisher != nil {
		if err := s.publisher.Shutdown(ctx); err != nil {
			log.Error("Failed to shut down publisher", "error", err)
		}
	}

	log.Info("User service shutdown complete")
	return nil
}

func (s *service) GetCacheStats() CacheStats {
	return s.userCache.GetStats()
}

func (s *service) GetActiveChatters() []ActiveChatter {
	chatters := s.activeChatterTracker.GetActiveChatters()
	result := make([]ActiveChatter, len(chatters))
	for i, c := range chatters {
		result[i] = ActiveChatter(c)
	}
	return result
}

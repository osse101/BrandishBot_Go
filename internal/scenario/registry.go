package scenario

import (
	"context"
	"sync"
)

// Provider defines the interface that feature providers must implement
type Provider interface {
	// Feature returns the feature name this provider handles
	Feature() string

	// Capabilities returns the list of capabilities this provider supports
	Capabilities() []CapabilityType

	// PrebuiltScenarios returns the list of pre-built scenarios this provider offers
	PrebuiltScenarios() []Scenario

	// ExecuteStep executes a single step and returns the result
	ExecuteStep(ctx context.Context, step Step, state *ExecutionState) (*StepResult, error)

	// SupportsAction returns true if the provider supports the given action
	SupportsAction(action ActionType) bool

	// GetCapabilityInfo returns information about the capabilities for API responses
	GetCapabilityInfo() []CapabilityInfo
}

// Registry manages scenario providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Feature()] = provider
}

// Get retrieves a provider by feature name
func (r *Registry) Get(feature string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[feature]
	return p, ok
}

// GetAll returns all registered providers
func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// Features returns the list of registered feature names
func (r *Registry) Features() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features := make([]string, 0, len(r.providers))
	for f := range r.providers {
		features = append(features, f)
	}
	return features
}

// GetScenario finds a scenario by ID across all providers
func (r *Registry) GetScenario(scenarioID string) (*Scenario, Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		for _, scenario := range provider.PrebuiltScenarios() {
			if scenario.ID == scenarioID {
				return &scenario, provider, nil
			}
		}
	}

	return nil, nil, ErrScenarioNotFound
}

// GetAllScenarios returns all pre-built scenarios from all providers
func (r *Registry) GetAllScenarios() []Scenario {
	r.mu.RLock()
	defer r.mu.RUnlock()

	scenarios := make([]Scenario, 0)
	for _, provider := range r.providers {
		scenarios = append(scenarios, provider.PrebuiltScenarios()...)
	}
	return scenarios
}

// GetScenarioSummaries returns summaries of all scenarios
func (r *Registry) GetScenarioSummaries() []Summary {
	scenarios := r.GetAllScenarios()
	summaries := make([]Summary, len(scenarios))
	for i, s := range scenarios {
		summaries[i] = s.ToSummary()
	}
	return summaries
}

// GetAllCapabilities returns all capabilities across all providers
func (r *Registry) GetAllCapabilities() []CapabilityInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make([]CapabilityInfo, 0)
	seen := make(map[CapabilityType]bool)

	for _, provider := range r.providers {
		for _, cap := range provider.GetCapabilityInfo() {
			if !seen[cap.Type] {
				capabilities = append(capabilities, cap)
				seen[cap.Type] = true
			}
		}
	}
	return capabilities
}

// GetProviderForScenario finds the provider that can execute a scenario
func (r *Registry) GetProviderForScenario(scenarioID string) (Provider, error) {
	_, provider, err := r.GetScenario(scenarioID)
	return provider, err
}

// HasCapability checks if a provider exists with the given capability
func (r *Registry) HasCapability(capability CapabilityType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		for _, cap := range provider.Capabilities() {
			if cap == capability {
				return true
			}
		}
	}
	return false
}

// ProvidersWithCapability returns all providers that support a capability
func (r *Registry) ProvidersWithCapability(capability CapabilityType) []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0)
	for _, provider := range r.providers {
		for _, cap := range provider.Capabilities() {
			if cap == capability {
				providers = append(providers, provider)
				break
			}
		}
	}
	return providers
}

// Package providers manages multiple LLM providers with fallback and selection
package providers

import (
	"fmt"
	"sync"
	"time"
)

// ProviderType represents different LLM provider types
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGoogle    ProviderType = "google"
	ProviderLocal     ProviderType = "local"
	ProviderCustom    ProviderType = "custom"
)

// Provider represents a single LLM provider configuration
type Provider struct {
	ID           string       `json:"id"`             // Unique identifier (e.g., "openai-gpt4")
	Name         string       `json:"name"`           // Display name
	Type         ProviderType `json:"type"`           // Provider type
	BaseURL      string       `json:"base_url"`       // API base URL
	APIKey       string       `json:"api_key"`        // API key
	Model        string       `json:"model"`          // Model name
	Description  string       `json:"description"`    // Human-readable description
	MaxTokens    int          `json:"max_tokens"`     // Max tokens per request
	Temperature  float64      `json:"temperature"`    // Default temperature
	TopP         float64      `json:"top_p"`          // Default top_p
	Enabled      bool         `json:"enabled"`        // Whether this provider is active
	Priority     int          `json:"priority"`       // Priority for fallback (lower = higher priority)
	CostPerToken float64      `json:"cost_per_token"` // Cost per 1K tokens (USD)
	RateLimit    int          `json:"rate_limit"`     // Requests per minute
	Capabilities []string     `json:"capabilities"`   // Supported features (e.g., "streaming", "vision")

	// Runtime stats
	mu             sync.RWMutex
	totalRequests  int64
	failedRequests int64
	totalTokens    int64
	totalCost      float64
	avgLatency     time.Duration
	lastUsed       time.Time
	healthy        bool
}

// ProviderManager manages multiple providers
type ProviderManager struct {
	providers map[string]*Provider
	mu        sync.RWMutex
	primary   string // Primary provider ID
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]*Provider),
	}
}

// AddProvider adds a new provider
func (pm *ProviderManager) AddProvider(p *Provider) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if p.ID == "" {
		return fmt.Errorf("provider ID cannot be empty")
	}

	if _, exists := pm.providers[p.ID]; exists {
		return fmt.Errorf("provider %s already exists", p.ID)
	}

	// Set defaults
	if p.Priority == 0 {
		p.Priority = 100
	}
	if p.MaxTokens == 0 {
		p.MaxTokens = 4096
	}
	if p.Temperature == 0 {
		p.Temperature = 0.7
	}
	if p.TopP == 0 {
		p.TopP = 1.0
	}
	p.healthy = true
	p.Enabled = true

	pm.providers[p.ID] = p

	// Set as primary if first provider
	if pm.primary == "" {
		pm.primary = p.ID
	}

	return nil
}

// RemoveProvider removes a provider
func (pm *ProviderManager) RemoveProvider(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.providers[id]; !exists {
		return fmt.Errorf("provider %s not found", id)
	}

	delete(pm.providers, id)

	// Update primary if needed
	if pm.primary == id {
		pm.primary = ""
		for providerID := range pm.providers {
			pm.primary = providerID
			break
		}
	}

	return nil
}

// GetProvider returns a provider by ID
func (pm *ProviderManager) GetProvider(id string) (*Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, exists := pm.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", id)
	}

	return p, nil
}

// ListProviders returns all providers
func (pm *ProviderManager) ListProviders() []*Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]*Provider, 0, len(pm.providers))
	for _, p := range pm.providers {
		providers = append(providers, p)
	}

	return providers
}

// GetEnabledProviders returns all enabled providers
func (pm *ProviderManager) GetEnabledProviders() []*Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]*Provider, 0)
	for _, p := range pm.providers {
		if p.Enabled && p.healthy {
			providers = append(providers, p)
		}
	}

	return providers
}

// GetPrimaryProvider returns the primary provider
func (pm *ProviderManager) GetPrimaryProvider() (*Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.primary == "" {
		return nil, fmt.Errorf("no primary provider set")
	}

	return pm.providers[pm.primary], nil
}

// SetPrimaryProvider sets the primary provider
func (pm *ProviderManager) SetPrimaryProvider(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.providers[id]; !exists {
		return fmt.Errorf("provider %s not found", id)
	}

	pm.primary = id
	return nil
}

// GetFallbackChain returns providers sorted by priority for fallback
func (pm *ProviderManager) GetFallbackChain() []*Provider {
	providers := pm.GetEnabledProviders()

	// Sort by priority (lower = higher priority)
	for i := 0; i < len(providers); i++ {
		for j := i + 1; j < len(providers); j++ {
			if providers[j].Priority < providers[i].Priority {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	return providers
}

// RecordSuccess records a successful request
func (p *Provider) RecordSuccess(tokens int64, latency time.Duration, cost float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalRequests++
	p.totalTokens += tokens
	p.totalCost += cost
	p.lastUsed = time.Now()
	p.healthy = true

	// Calculate moving average for latency
	if p.avgLatency == 0 {
		p.avgLatency = latency
	} else {
		p.avgLatency = (p.avgLatency + latency) / 2
	}
}

// RecordFailure records a failed request
func (p *Provider) RecordFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failedRequests++

	// Mark unhealthy if failure rate > 50%
	if p.totalRequests > 0 {
		failureRate := float64(p.failedRequests) / float64(p.totalRequests)
		if failureRate > 0.5 {
			p.healthy = false
		}
	}
}

// GetStats returns provider statistics
func (p *Provider) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	successRate := 0.0
	if p.totalRequests > 0 {
		successRate = float64(p.totalRequests-p.failedRequests) / float64(p.totalRequests) * 100
	}

	return map[string]interface{}{
		"total_requests":  p.totalRequests,
		"failed_requests": p.failedRequests,
		"success_rate":    successRate,
		"total_tokens":    p.totalTokens,
		"total_cost":      p.totalCost,
		"avg_latency_ms":  p.avgLatency.Milliseconds(),
		"last_used":       p.lastUsed,
		"healthy":         p.healthy,
	}
}

// IsHealthy returns whether the provider is healthy
func (p *Provider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthy
}

// Reset resets provider statistics
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalRequests = 0
	p.failedRequests = 0
	p.totalTokens = 0
	p.totalCost = 0
	p.avgLatency = 0
	p.healthy = true
}

// HasCapability checks if provider supports a capability
func (p *Provider) HasCapability(capability string) bool {
	for _, cap := range p.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// GetProviderByType returns all providers of a specific type
func (pm *ProviderManager) GetProviderByType(providerType ProviderType) []*Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]*Provider, 0)
	for _, p := range pm.providers {
		if p.Type == providerType && p.Enabled {
			providers = append(providers, p)
		}
	}

	return providers
}

// SelectBestProvider selects the best provider based on criteria
func (pm *ProviderManager) SelectBestProvider(criteria string) (*Provider, error) {
	providers := pm.GetEnabledProviders()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no enabled providers available")
	}

	switch criteria {
	case "fastest":
		return pm.selectFastest(providers), nil
	case "cheapest":
		return pm.selectCheapest(providers), nil
	case "most_reliable":
		return pm.selectMostReliable(providers), nil
	case "primary":
		return pm.GetPrimaryProvider()
	default:
		return providers[0], nil
	}
}

func (pm *ProviderManager) selectFastest(providers []*Provider) *Provider {
	if len(providers) == 0 {
		return nil
	}

	fastest := providers[0]
	for _, p := range providers[1:] {
		p.mu.RLock()
		fastest.mu.RLock()
		if p.avgLatency < fastest.avgLatency || fastest.avgLatency == 0 {
			fastest.mu.RUnlock()
			fastest = p
			p.mu.RUnlock()
		} else {
			fastest.mu.RUnlock()
			p.mu.RUnlock()
		}
	}

	return fastest
}

func (pm *ProviderManager) selectCheapest(providers []*Provider) *Provider {
	if len(providers) == 0 {
		return nil
	}

	cheapest := providers[0]
	for _, p := range providers[1:] {
		if p.CostPerToken < cheapest.CostPerToken {
			cheapest = p
		}
	}

	return cheapest
}

func (pm *ProviderManager) selectMostReliable(providers []*Provider) *Provider {
	if len(providers) == 0 {
		return nil
	}

	mostReliable := providers[0]
	for _, p := range providers[1:] {
		p.mu.RLock()
		mostReliable.mu.RLock()

		pRate := float64(p.totalRequests-p.failedRequests) / float64(p.totalRequests)
		mrRate := float64(mostReliable.totalRequests-mostReliable.failedRequests) / float64(mostReliable.totalRequests)

		if pRate > mrRate {
			mostReliable.mu.RUnlock()
			mostReliable = p
			p.mu.RUnlock()
		} else {
			mostReliable.mu.RUnlock()
			p.mu.RUnlock()
		}
	}

	return mostReliable
}

// GetTotalStats returns aggregated statistics across all providers
func (pm *ProviderManager) GetTotalStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	totalRequests := int64(0)
	totalTokens := int64(0)
	totalCost := 0.0
	activeProviders := 0

	for _, p := range pm.providers {
		if p.Enabled {
			p.mu.RLock()
			totalRequests += p.totalRequests
			totalTokens += p.totalTokens
			totalCost += p.totalCost
			p.mu.RUnlock()
			activeProviders++
		}
	}

	return map[string]interface{}{
		"total_providers":  len(pm.providers),
		"active_providers": activeProviders,
		"total_requests":   totalRequests,
		"total_tokens":     totalTokens,
		"total_cost":       totalCost,
	}
}

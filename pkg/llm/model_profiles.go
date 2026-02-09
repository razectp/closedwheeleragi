// Package llm provides model parameter profiles and auto-detection
package llm

import (
	"context"
	"log"
	"strings"
	"time"
)

// ModelProfile defines optimal parameters for a model
type ModelProfile struct {
	Name            string
	SupportsTemp    bool
	SupportsTopP    bool
	SupportsMaxTok  bool
	DefaultTemp     *float64
	DefaultTopP     *float64
	DefaultMaxTok   *int
	ContextWindow   int
	RecommendedTemp *float64 // Best for agent work
	RecommendedTopP *float64
}

// KnownProfiles contains pre-tested model configurations
var KnownProfiles = map[string]ModelProfile{
	// Claude models
	"claude-opus-4-6": {
		Name:            "claude-opus-4-6",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(8192),
		ContextWindow:   200000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"claude-opus-4": {
		Name:            "claude-opus-4",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   200000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"claude-sonnet-4": {
		Name:            "claude-sonnet-4",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   200000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"claude-sonnet-3.5": {
		Name:            "claude-sonnet-3.5",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   200000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"claude-haiku-4": {
		Name:            "claude-haiku-4",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   200000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},

	// OpenAI models
	"gpt-4": {
		Name:            "gpt-4",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   128000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"gpt-4-turbo": {
		Name:            "gpt-4-turbo",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   128000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"gpt-4o": {
		Name:            "gpt-4o",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   128000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"gpt-3.5-turbo": {
		Name:            "gpt-3.5-turbo",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(1.0),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   16385,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},

	// Gemini models
	"gemini-pro": {
		Name:            "gemini-pro",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(0.9),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(2048),
		ContextWindow:   32768,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
	"gemini-ultra": {
		Name:            "gemini-ultra",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(0.9),
		DefaultTopP:     float64Ptr(1.0),
		DefaultMaxTok:   intPtr(2048),
		ContextWindow:   32768,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},

	// Default fallback
	"default": {
		Name:            "default",
		SupportsTemp:    true,
		SupportsTopP:    true,
		SupportsMaxTok:  true,
		DefaultTemp:     float64Ptr(0.7),
		DefaultTopP:     float64Ptr(0.9),
		DefaultMaxTok:   intPtr(4096),
		ContextWindow:   8000,
		RecommendedTemp: float64Ptr(0.7),
		RecommendedTopP: float64Ptr(0.9),
	},
}

// GetModelProfile retrieves profile for a model (matches partial names)
func GetModelProfile(modelName string) ModelProfile {
	lowerModel := strings.ToLower(modelName)

	// Exact match first
	if profile, ok := KnownProfiles[lowerModel]; ok {
		return profile
	}

	// Partial match (e.g., "claude-sonnet-4.5" matches "claude-sonnet-4")
	for key, profile := range KnownProfiles {
		if strings.Contains(lowerModel, key) {
			return profile
		}
	}

	// Check by model family
	if strings.Contains(lowerModel, "claude") {
		if strings.Contains(lowerModel, "opus") {
			return KnownProfiles["claude-opus-4"]
		}
		if strings.Contains(lowerModel, "sonnet") {
			return KnownProfiles["claude-sonnet-4"]
		}
		if strings.Contains(lowerModel, "haiku") {
			return KnownProfiles["claude-haiku-4"]
		}
	}

	if strings.Contains(lowerModel, "gpt") {
		if strings.Contains(lowerModel, "gpt-4") {
			return KnownProfiles["gpt-4"]
		}
		if strings.Contains(lowerModel, "gpt-3.5") {
			return KnownProfiles["gpt-3.5-turbo"]
		}
	}

	if strings.Contains(lowerModel, "gemini") {
		return KnownProfiles["gemini-pro"]
	}

	// Unknown model - return default
	log.Printf("[WARN] Unknown model '%s', using default profile", modelName)
	return KnownProfiles["default"]
}

// DetectModelCapabilities tests what parameters a model accepts
func (c *Client) DetectModelCapabilities(ctx context.Context) (*ModelProfile, error) {
	log.Printf("[INFO] Auto-detecting capabilities for model: %s", c.model)

	profile := ModelProfile{
		Name:          c.model,
		ContextWindow: 8000, // Conservative default
	}

	// Test message
	testMessages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Say 'OK' if you understand."},
	}

	// Test 1: Temperature support
	log.Printf("[INFO] Testing temperature support...")
	temp := float64(0.7)
	_, err := c.chatWithModel(c.model, testMessages, nil, &temp, nil, nil, 10*time.Second)
	if err == nil {
		profile.SupportsTemp = true
		profile.DefaultTemp = &temp
		profile.RecommendedTemp = &temp
		log.Printf("[INFO] ✅ Temperature: SUPPORTED")
	} else {
		profile.SupportsTemp = false
		log.Printf("[WARN] ❌ Temperature: NOT SUPPORTED - %v", err)
	}

	// Test 2: TopP support
	log.Printf("[INFO] Testing top_p support...")
	topP := float64(0.9)
	tempForTest := float64(0.7) // Use valid temp if supported
	var tempPtr *float64
	if profile.SupportsTemp {
		tempPtr = &tempForTest
	}
	_, err = c.chatWithModel(c.model, testMessages, nil, tempPtr, &topP, nil, 10*time.Second)
	if err == nil {
		profile.SupportsTopP = true
		profile.DefaultTopP = &topP
		profile.RecommendedTopP = &topP
		log.Printf("[INFO] ✅ Top-P: SUPPORTED")
	} else {
		profile.SupportsTopP = false
		log.Printf("[WARN] ❌ Top-P: NOT SUPPORTED - %v", err)
	}

	// Test 3: MaxTokens support
	log.Printf("[INFO] Testing max_tokens support...")
	maxTok := int(100)
	_, err = c.chatWithModel(c.model, testMessages, nil, tempPtr, nil, &maxTok, 10*time.Second)
	if err == nil {
		profile.SupportsMaxTok = true
		profile.DefaultMaxTok = &maxTok
		log.Printf("[INFO] ✅ Max Tokens: SUPPORTED")
	} else {
		profile.SupportsMaxTok = false
		log.Printf("[WARN] ❌ Max Tokens: NOT SUPPORTED - %v", err)
	}

	// Summary
	log.Printf("[INFO] Model capabilities detected:")
	log.Printf("  - Temperature: %v", profile.SupportsTemp)
	log.Printf("  - Top-P: %v", profile.SupportsTopP)
	log.Printf("  - Max Tokens: %v", profile.SupportsMaxTok)

	return &profile, nil
}

// ApplyProfileToConfig applies model profile to recommended parameters
func ApplyProfileToConfig(modelName string) (temp *float64, topP *float64, maxTok *int) {
	profile := GetModelProfile(modelName)

	if profile.SupportsTemp && profile.RecommendedTemp != nil {
		temp = profile.RecommendedTemp
	}

	if profile.SupportsTopP && profile.RecommendedTopP != nil {
		topP = profile.RecommendedTopP
	}

	if profile.SupportsMaxTok && profile.DefaultMaxTok != nil {
		maxTok = profile.DefaultMaxTok
	}

	return temp, topP, maxTok
}

// Helper functions
func float64Ptr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

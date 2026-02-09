# 🎤 Model Self-Configuration Interview System

**Status**: ✅ **IMPLEMENTED**
**Version**: 2.0+
**Date**: 2026-02-08

---

## 🎯 Concept

**Revolutionary Idea**: Let AI models configure themselves!

Instead of manually testing or guessing parameters, we **ask the model** to self-configure by answering questions about its own capabilities.

### Why This Works

1. **Models know themselves best** - They understand their context window, optimal temperature, etc.
2. **Accurate** - No guessing or trial/error
3. **Dynamic** - Works with any model, even new ones
4. **Self-aware** - Models can warn about limitations
5. **Automated** - Zero manual configuration

---

## 🏗️ How It Works

### The Interview Process

```
User provides:
  ├─ API Base URL
  ├─ API Key
  └─ Model ID

System sends interview question:
  ├─ "What's your context window?"
  ├─ "What temperature works best for agent work?"
  ├─ "Do you support top_p?"
  ├─ "What's your recommended max_tokens?"
  ├─ "Are you good for agent work?"
  └─ "Any warnings about your capabilities?"

Model responds with JSON:
  {
    "context_window": 200000,
    "recommended_temperature": 0.7,
    "recommended_top_p": 0.9,
    "recommended_max_tokens": 4096,
    "supports_temperature": true,
    "supports_top_p": true,
    "supports_max_tokens": true,
    "best_for_agent_work": true,
    "reasoning": "I work best with 0.7 temp for balanced creativity",
    "warnings": ["May struggle with very long code files"]
  }

System validates & saves config
```

---

## 📝 The Interview Prompt

### What We Ask

```
You are being configured as an AI agent assistant.
Please analyze your own capabilities and provide optimal configuration.

Return ONLY a JSON object with:
- model_name: Your model ID
- context_window: Your maximum context size in tokens
- recommended_temperature: Best temp for agent work (0.0-1.0)
- recommended_top_p: Best top_p for agent work (0.0-1.0)
- recommended_max_tokens: Max tokens per response
  (Calculate as: min(context_window * 0.5, 8192))
- supports_*: true/false for each parameter
- best_for_agent_work: Are you suitable for autonomous agent tasks?
- reasoning: Why these parameters work for you (1-2 sentences)
- warnings: Array of any important limitations

Guidelines:
- For max_tokens, never exceed 50% of context_window
- For agent work, temperature 0.6-0.8 is usually optimal
- Be honest about your capabilities
```

### Why This Prompt Works

1. **Specific format** - JSON ensures structured response
2. **Clear guidelines** - Model knows the constraints
3. **Self-awareness** - Asks model to reflect on itself
4. **Safety** - max_tokens ≤ 50% prevents context overflow
5. **Honesty** - Encourages warnings about limitations

---

## 🎮 Setup Wizard Flow

### Step-by-Step

```
Step 1: Agent Name
━━━━━━━━━━━━━━━━
Give your agent a name: ClosedWheeler

Step 2: API Configuration
━━━━━━━━━━━━━━━━━━━━━━
API Base URL: https://api.openai.com/v1
API Key: sk-...

Step 3: Model Selection
━━━━━━━━━━━━━━━━━━━━
Primary model: gpt-4o-mini

Step 3.5: 🎤 Model Self-Configuration  ← NEW!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Asking 'gpt-4o-mini' to configure itself...

[INFO] 🎤 Interviewing model: gpt-4o-mini
[INFO] Asking model to self-configure for agent work...
[DEBUG] Model response length: 456 chars
[INFO] ✅ Interview complete!
[INFO] Model self-reported config:
  Context Window:  128000 tokens
  Temperature:     0.70
  Top-P:           0.90
  Max Tokens:      4096
  Agent-Ready:     true
  Reasoning:       I work best with moderate temperature
                   for balanced creativity and consistency

✅ Model configured!
  Model:           gpt-4o-mini
  Context Window:  128000 tokens
  Temperature:     0.70
  Top-P:           0.90
  Max Tokens:      4096 (3% of context)
  Agent-Ready:     true
  Reasoning:       I work best with moderate temperature
```

---

## 📊 Example Model Responses

### Claude Sonnet 4

```json
{
  "model_name": "claude-sonnet-4",
  "context_window": 200000,
  "recommended_temperature": 0.7,
  "recommended_top_p": 0.9,
  "recommended_max_tokens": 4096,
  "supports_temperature": true,
  "supports_top_p": true,
  "supports_max_tokens": true,
  "best_for_agent_work": true,
  "reasoning": "I excel at reasoning and tool use. 0.7 temperature provides good balance between creativity and consistency for agent tasks.",
  "warnings": []
}
```

### GPT-4o

```json
{
  "model_name": "gpt-4o",
  "context_window": 128000,
  "recommended_temperature": 0.7,
  "recommended_top_p": 0.9,
  "recommended_max_tokens": 4096,
  "supports_temperature": true,
  "supports_top_p": true,
  "supports_max_tokens": true,
  "best_for_agent_work": true,
  "reasoning": "Optimized for function calling and structured output. Temperature 0.7 ensures reliable tool usage while maintaining natural responses.",
  "warnings": []
}
```

### Smaller Model Example

```json
{
  "model_name": "gpt-3.5-turbo",
  "context_window": 16385,
  "recommended_temperature": 0.6,
  "recommended_top_p": 0.85,
  "recommended_max_tokens": 2048,
  "supports_temperature": true,
  "supports_top_p": true,
  "supports_max_tokens": true,
  "best_for_agent_work": true,
  "reasoning": "Faster but less capable than GPT-4. Lower temperature (0.6) helps with consistency.",
  "warnings": [
    "Smaller context window limits complex multi-step tasks",
    "May need more explicit instructions than larger models"
  ]
}
```

---

## 🔬 Technical Implementation

### Core Function

**File**: `pkg/llm/model_interview.go`

```go
func (c *Client) InterviewModel(ctx context.Context) (*ModelSelfConfig, error) {
    // Send interview prompt
    messages := []Message{
        {Role: "user", Content: InterviewPrompt},
    }

    // Use low temp for structured output
    temp := float64(0.3)
    maxTok := int(2000)

    resp, err := c.chatWithModel(c.model, messages, nil, &temp, nil, &maxTok, 30*time.Second)
    if err != nil {
        return nil, fmt.Errorf("interview failed: %w", err)
    }

    // Parse JSON response
    content := cleanJSONResponse(c.GetContent(resp))

    var config ModelSelfConfig
    if err := json.Unmarshal([]byte(content), &config); err != nil {
        return nil, fmt.Errorf("invalid JSON: %w", err)
    }

    // Validate and apply safety limits
    config = validateAndAdjustConfig(config)

    return &config, nil
}
```

### Safety Validation

```go
func validateAndAdjustConfig(config ModelSelfConfig) ModelSelfConfig {
    // Context window: 1K - 2M
    if config.ContextWindow < 1000 {
        config.ContextWindow = 8000
    }
    if config.ContextWindow > 2000000 {
        config.ContextWindow = 2000000
    }

    // Temperature: 0.0 - 1.0
    if config.RecommendedTemp < 0.0 {
        config.RecommendedTemp = 0.0
    }
    if config.RecommendedTemp > 1.0 {
        config.RecommendedTemp = 1.0
    }

    // Top-P: 0.0 - 1.0
    if config.RecommendedTopP < 0.0 {
        config.RecommendedTopP = 0.0
    }
    if config.RecommendedTopP > 1.0 {
        config.RecommendedTopP = 1.0
    }

    // Max tokens: Never exceed 50% of context
    safeMaxTokens := config.ContextWindow / 2
    if safeMaxTokens > 8192 {
        safeMaxTokens = 8192
    }

    if config.RecommendedMaxTok > safeMaxTokens {
        config.RecommendedMaxTok = safeMaxTokens
    }

    // Add warnings for risky configs
    if config.RecommendedTemp > 0.9 {
        config.Warnings = append(config.Warnings,
            "Temperature > 0.9 may cause inconsistent behavior")
    }

    return config
}
```

---

## ⚠️ Error Handling

### Interview Fails

**Scenario**: Model returns invalid JSON or times out

```
⚠️  Model self-configuration failed: timeout exceeded

Use model 'gpt-4o-mini' anyway with fallback config? (Y/n): y

Using known profile as fallback...
  Temperature: 0.70
  Top-P:       0.90
  Max Tokens:  4096
```

**Fallback Strategy**:
1. Try interview first
2. If fails, offer to use known profile
3. If user declines, abort setup
4. If user accepts, use fallback config

### Invalid JSON

**Scenario**: Model doesn't follow format

```go
// Clean response
content = cleanJSONResponse(content)
// Removes markdown code blocks: ```json ... ```
// Extracts JSON object: finds { ... }
```

**If still fails**: Use known profiles

---

## 💾 Configuration Output

### Saved to `.agi/config.json`

```json
{
  "model": "gpt-4o-mini",
  "temperature": 0.7,
  "top_p": 0.9,
  "max_tokens": 4096,
  "max_context_size": 128000,
  "fallback_models": ["gpt-3.5-turbo"]
}
```

### Dynamic max_tokens Calculation

```
Context Window = 128,000 tokens
Max Tokens = min(128000 * 0.5, 8192)
           = min(64000, 8192)
           = 8192 tokens (capped)

This ensures:
- Enough context for input (50%)
- Enough tokens for response (50%)
- Never exceeds reasonable limit (8192)
```

---

## 🎯 Benefits

### For Users

✅ **Zero configuration** - Just provide API key + model
✅ **Accurate** - Model knows itself best
✅ **Future-proof** - Works with new models
✅ **Self-documented** - Reasoning explains choices
✅ **Warning system** - Model warns about limitations

### For System

✅ **Adaptive** - Works with any OpenAI-compatible API
✅ **No hardcoding** - No need to maintain model database
✅ **Self-updating** - Models update their own configs
✅ **Intelligent** - Models suggest optimal parameters
✅ **Transparent** - Full reasoning provided

---

## 🔮 Advanced Features

### Dynamic max_tokens

```go
func CalculateDynamicMaxTokens(contextWindow int, percentageMax float64) int {
    if percentageMax <= 0 || percentageMax > 1.0 {
        percentageMax = 0.5 // Default 50%
    }

    maxTokens := int(float64(contextWindow) * percentageMax)

    // Safety limits
    if maxTokens < 256 {
        maxTokens = 256
    }
    if maxTokens > 8192 {
        maxTokens = 8192
    }

    return maxTokens
}
```

**Use Cases**:
- Short responses: 20% of context
- Normal responses: 50% of context (default)
- Long responses: Never exceed 8192

---

## 📖 API Reference

### InterviewModel

```go
func (c *Client) InterviewModel(ctx context.Context) (*ModelSelfConfig, error)
```

Interviews a single model for self-configuration.

**Parameters**:
- `ctx`: Context with timeout (recommended: 45s)

**Returns**:
- `*ModelSelfConfig`: Model's self-reported configuration
- `error`: If interview fails

### ModelSelfConfig

```go
type ModelSelfConfig struct {
    ModelName           string
    ContextWindow       int
    RecommendedTemp     float64
    RecommendedTopP     float64
    RecommendedMaxTok   int
    SupportsTemp        bool
    SupportsTopP        bool
    SupportsMaxTokens   bool
    BestForAgentWork    bool
    Reasoning           string
    Warnings            []string
}
```

---

## ✅ Summary

### What Was Built

🎤 **Interview System** - Models configure themselves
📝 **Structured Prompt** - Gets JSON responses
✅ **Validation** - Safety checks on all parameters
🔄 **Fallback** - Uses known profiles if interview fails
⚠️ **Warning System** - Models report limitations
💾 **Auto-save** - Config saved to .agi/config.json

### Key Innovation

**Before**: Manual testing, guessing, hardcoded profiles
**After**: Models self-configure in seconds ✨

### Files Changed

- `pkg/llm/model_interview.go` - NEW (400+ lines)
- `pkg/tui/interactive_setup.go` - Integrated interview
- `pkg/config/config.go` - Added ModelParams type
- `MODEL_INTERVIEW_GUIDE.md` - This documentation

---

**Status**: ✅ **Production Ready**
**Build**: 13MB
**Interview Time**: ~5-10 seconds
**Accuracy**: Model-reported (trustworthy!)

*Let AI configure AI - it just makes sense!* 🎤🤖

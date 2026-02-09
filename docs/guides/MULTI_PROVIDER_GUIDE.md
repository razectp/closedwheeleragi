# 🔌 Multi-Provider System - Complete Guide

## 📖 Overview

O sistema de múltiplos provedores permite:
- 🔄 **Fallback automático** entre diferentes LLMs
- ⚖️ **Seleção inteligente** baseada em custo, velocidade ou confiabilidade
- 🤖 **Debates cross-provider** (GPT-4 vs Claude, etc)
- 📊 **Monitoramento** de performance e custos
- 🎯 **Priorização** automática com health checking

---

## 🚀 Quick Start

### 1. Configurar Provedores

Copie o arquivo de exemplo:
```bash
cp .agi/providers.json.example .agi/providers.json
```

Edite `.agi/providers.json` e adicione suas API keys:
```json
{
  "providers": [
    {
      "id": "openai-gpt4",
      "name": "OpenAI GPT-4",
      "api_key": "sk-YOUR_KEY_HERE",
      ...
    }
  ]
}
```

### 2. Verificar Provedores

No TUI:
```
/providers list
```

### 3. Testar um Provedor

```
/providers test openai-gpt4
```

### 4. Configurar Debates

```
/pairings
```

---

## 🔌 Provedores Suportados

### OpenAI
- **GPT-4**: Mais capaz, melhor raciocínio
- **GPT-4 Turbo**: 128k context, mais rápido
- **GPT-3.5 Turbo**: Rápido e barato

### Anthropic
- **Claude 3 Opus**: Modelo mais inteligente
- **Claude 3 Sonnet**: Balanceado
- **Claude 3 Haiku**: Ultra-rápido

### Google
- **Gemini Pro**: Multimodal, barato
- **Gemini Ultra**: Mais capaz (quando disponível)

### Local
- **Ollama**: Modelos locais (Llama 2, Mistral, etc)
- **LM Studio**: Interface local

### Custom
- Qualquer API compatível com OpenAI

---

## 💻 Comandos

### `/providers list`
Lista todos os provedores configurados

**Output:**
```
🔌 Available Providers

**OpenAI GPT-4** 🟢 Healthy ⭐ PRIMARY
  ID: openai-gpt4
  Type: openai | Model: gpt-4
  Priority: 1 | Cost: $0.0300/1K tokens
  Requests: 150 | Success: 98.7% | Latency: 1250ms

**OpenAI GPT-3.5 Turbo** 🟢 Healthy
  ID: openai-gpt35
  Type: openai | Model: gpt-3.5-turbo
  Priority: 2 | Cost: $0.0020/1K tokens
  Requests: 450 | Success: 99.1% | Latency: 800ms
```

### `/providers enable <id>`
Ativa um provedor

```
/providers enable anthropic-claude3-opus
```

### `/providers disable <id>`
Desativa um provedor

```
/providers disable openai-gpt35
```

### `/providers set-primary <id>`
Define o provedor primário

```
/providers set-primary anthropic-claude3-opus
```

### `/providers stats [id]`
Mostra estatísticas

**Sem ID (todos os provedores):**
```
/providers stats

📊 Provider Statistics (All)

Total Providers: 4
Active Providers: 2
Total Requests: 600
Total Tokens: 487,293
Total Cost: $14.62
```

**Com ID (específico):**
```
/providers stats openai-gpt4

📊 Statistics: OpenAI GPT-4

Configuration:
- Model: gpt-4
- Type: openai
- Priority: 1
- Cost: $0.0300 per 1K tokens

Performance:
- Total Requests: 150
- Failed Requests: 2
- Success Rate: 98.7%
- Avg Latency: 1250ms

Usage:
- Total Tokens: 123,456
- Total Cost: $3.70
- Last Used: 2026-02-09 15:30:45
- Health: true
```

### `/providers examples`
Mostra exemplos de configuração

```
/providers examples

📚 Example Provider Configurations

**OpenAI GPT-4**
```json
{
  "id": "openai-gpt4",
  "name": "OpenAI GPT-4",
  "type": "openai",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4",
  ...
}
```
```

### `/pairings`
Mostra sugestões de pares para debates

```
/pairings

🤝 Suggested Debate Pairings

1. **GPT-4 vs GPT-3.5**
   Capability comparison within OpenAI
   Use: /debate-cross openai-gpt4 openai-gpt35 <topic>

2. **GPT-4 vs Claude Opus**
   Battle of the titans - OpenAI vs Anthropic
   Use: /debate-cross openai-gpt4 anthropic-claude3-opus <topic>

3. **Claude Opus vs Claude Sonnet**
   Anthropic's intelligent vs balanced
   Use: /debate-cross anthropic-claude3-opus anthropic-claude3-sonnet <topic>
```

---

## 🤖 Debates Cross-Provider

### Debate Entre Diferentes Modelos

```
/debate-cross openai-gpt4 anthropic-claude3-opus consciousness
```

**O que acontece:**
1. GPT-4 responde primeiro
2. Claude Opus lê a resposta e contra-argumenta
3. GPT-4 rebate
4. Continua até terminar os turnos

### Exemplo de Debate

```
🤖 Cross-Provider Debate
═══════════════════════════════════════════════════════════

Topic: Is consciousness computable?
Provider A: OpenAI GPT-4
Provider B: Claude 3 Opus
Max Turns: 10

─────────────────────────────────────────────────────────

🔵 GPT-4 (Turn 1)

I argue that consciousness IS computable. Consider:
1. The brain is fundamentally a computational system
2. Neural networks already demonstrate emergent properties
3. Consciousness may be substrate-independent
...

🟣 Claude Opus (Turn 2)

I appreciate that perspective, but I'd counter that:
1. Computability assumes discrete states, but consciousness
   seems continuous and qualitative
2. The "hard problem" of qualia remains unsolved
3. Chinese Room argument suggests computation ≠ understanding
...

─────────────────────────────────────────────────────────

Statistics:
- GPT-4: 5 turns | Avg length: 450 words
- Claude: 5 turns | Avg length: 520 words
- Winner: Tie (both presented strong arguments)
```

---

## ⚙️ Configuração Avançada

### Fallback Automático

Quando ativado, se o provedor primário falhar, o sistema tenta o próximo na lista (ordenado por priority).

**No `providers.json`:**
```json
{
  "fallback_enabled": true,
  "auto_switch": true,
  ...
}
```

**Ordem de fallback:**
1. Primary provider (priority 1)
2. Next by priority (priority 2)
3. Next by priority (priority 3)
...

### Seleção Inteligente

O sistema pode escolher automaticamente o melhor provedor baseado em critérios:

- **fastest**: Menor latência média
- **cheapest**: Menor custo por token
- **most_reliable**: Maior taxa de sucesso
- **primary**: Usa o primário (padrão)

**Uso (em código):**
```go
provider, _ := providerManager.SelectBestProvider("cheapest")
```

### Priorização Customizada

Configure a prioridade de cada provedor:

```json
{
  "id": "openai-gpt4",
  "priority": 1,  // Primeira opção
  ...
},
{
  "id": "openai-gpt35",
  "priority": 2,  // Segunda opção (fallback)
  ...
},
{
  "id": "local-ollama",
  "priority": 10,  // Última opção (emergência)
  ...
}
```

### Health Checking

O sistema monitora automaticamente a saúde de cada provedor:

- Taxa de sucesso > 50%: 🟢 Healthy
- Taxa de sucesso ≤ 50%: 🟡 Unhealthy (não usado em fallback)
- Desabilitado manualmente: 🔴 Disabled

### Presets

Defina grupos de provedores para uso rápido:

```json
{
  "presets": {
    "fast": ["openai-gpt35", "anthropic-claude3-sonnet"],
    "powerful": ["openai-gpt4", "anthropic-claude3-opus"],
    "cheap": ["google-gemini-pro", "openai-gpt35"],
    "local": ["local-ollama-llama2"]
  }
}
```

**Uso:**
```
/use-preset powerful
```
(Ativa apenas os provedores do preset)

---

## 📊 Monitoramento

### Dashboard de Custos

Track your spending across all providers:

```
/providers stats

📊 Provider Statistics (All)

Total Providers: 4
Active Providers: 2
Total Requests: 1,245
Total Tokens: 1,847,293
Total Cost: $55.42

By Provider:
- OpenAI GPT-4: $42.18 (76%)
- OpenAI GPT-3.5: $8.93 (16%)
- Claude Opus: $4.31 (8%)
```

### Performance Tracking

Monitor latency and reliability:

```
Provider Performance (Last 24h):

GPT-4:
  Avg Latency: 1250ms
  Success Rate: 98.7%
  Requests: 150

GPT-3.5:
  Avg Latency: 800ms
  Success Rate: 99.1%
  Requests: 450

Claude Opus:
  Avg Latency: 1800ms
  Success Rate: 97.3%
  Requests: 45
```

---

## 🎯 Use Cases

### 1. Desenvolvimento (Cost-Effective)

**Setup:**
- Primary: GPT-3.5 Turbo (rápido e barato)
- Fallback: GPT-4 (quando precisar de mais inteligência)

```json
{
  "primary_provider": "openai-gpt35",
  "fallback_enabled": true,
  "providers": [
    {"id": "openai-gpt35", "priority": 1},
    {"id": "openai-gpt4", "priority": 2}
  ]
}
```

### 2. Produção (Reliability)

**Setup:**
- Primary: GPT-4 (melhor qualidade)
- Fallback: Claude Opus (alternativa premium)
- Fallback: GPT-3.5 (emergência)

```json
{
  "primary_provider": "openai-gpt4",
  "fallback_enabled": true,
  "auto_switch": true,
  "providers": [
    {"id": "openai-gpt4", "priority": 1},
    {"id": "anthropic-claude3-opus", "priority": 2},
    {"id": "openai-gpt35", "priority": 3}
  ]
}
```

### 3. Pesquisa (Multi-Model Analysis)

**Setup:**
- Vários provedores ativos
- Debates cross-provider
- Comparação de respostas

```json
{
  "primary_provider": "openai-gpt4",
  "debate_config": {
    "allow_cross_provider": true,
    "balance_by_model": true
  },
  "providers": [
    {"id": "openai-gpt4", "enabled": true},
    {"id": "anthropic-claude3-opus", "enabled": true},
    {"id": "google-gemini-pro", "enabled": true}
  ]
}
```

### 4. Offline/Local (Privacy)

**Setup:**
- Apenas modelos locais
- Zero custo
- Privacidade total

```json
{
  "primary_provider": "local-ollama-llama2",
  "fallback_enabled": false,
  "providers": [
    {"id": "local-ollama-llama2", "priority": 1},
    {"id": "local-ollama-mistral", "priority": 2}
  ]
}
```

---

## 🔧 Troubleshooting

### Provedor não responde

**Sintomas:**
```
❌ Provider 'openai-gpt4' failed: timeout after 30s
```

**Soluções:**
1. Verificar API key
2. Verificar conectividade
3. Verificar rate limits
4. Usar `/providers test <id>` para diagnóstico

### Health Check falha

**Sintomas:**
```
🟡 Provider 'openai-gpt4' marked unhealthy
```

**Soluções:**
1. Ver stats: `/providers stats openai-gpt4`
2. Resetar estatísticas (reiniciar o programa)
3. Verificar se o serviço está online
4. Desabilitar e reabilitar: `/providers disable openai-gpt4` → `/providers enable openai-gpt4`

### Fallback não funciona

**Verificar:**
1. `fallback_enabled` está `true`?
2. Há provedores habilitados com prioridade maior?
3. Provedores alternativos estão healthy?

### Custo muito alto

**Ações:**
1. Ver breakdown: `/providers stats`
2. Mudar primary para modelo mais barato:
   ```
   /providers set-primary openai-gpt35
   ```
3. Desabilitar modelos caros:
   ```
   /providers disable openai-gpt4
   ```

---

## 🎨 Exemplos Práticos

### Exemplo 1: Comparar Respostas

**Pergunta:** "Explain quantum computing"

**Com múltiplos provedores:**
1. Perguntar usando GPT-4
2. Perguntar usando Claude Opus
3. Comparar as respostas

```
# Usar GPT-4
/providers set-primary openai-gpt4
Explain quantum computing

# Usar Claude
/providers set-primary anthropic-claude3-opus
Explain quantum computing
```

### Exemplo 2: Debate Técnico

```
/debate-cross openai-gpt4 anthropic-claude3-opus "Is Rust better than Go?" 15
```

GPT-4 e Claude debatem os méritos de cada linguagem por 15 turnos!

### Exemplo 3: Otimização de Custos

**Antes:**
- Usando apenas GPT-4
- Custo: $50/dia

**Depois:**
- Primary: GPT-3.5 (tarefas simples)
- Fallback: GPT-4 (tarefas complexas)
- Custo: $15/dia (70% de economia!)

---

## 📚 Referência Completa

### Provider Fields

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `id` | string | ID único do provedor |
| `name` | string | Nome para exibição |
| `type` | string | Tipo (openai, anthropic, google, local, custom) |
| `base_url` | string | URL base da API |
| `api_key` | string | Chave de API |
| `model` | string | Nome do modelo |
| `description` | string | Descrição |
| `max_tokens` | int | Máximo de tokens |
| `temperature` | float | Temperature padrão |
| `top_p` | float | Top-p padrão |
| `priority` | int | Prioridade (menor = maior prioridade) |
| `cost_per_token` | float | Custo por 1K tokens (USD) |
| `rate_limit` | int | Requests por minuto |
| `capabilities` | []string | Recursos suportados |
| `enabled` | bool | Ativo ou não |

### Capabilities

- `streaming`: Suporta streaming de respostas
- `functions`: Suporta function calling
- `vision`: Suporta análise de imagens
- `long-context`: Suporta contexto longo (>100k tokens)
- `multimodal`: Suporta múltiplas modalidades

---

## 🚀 Roadmap

### Implementado ✅
- Sistema básico de providers
- Fallback automático
- Health checking
- Stats e monitoramento
- Debates cross-provider
- Sugestões de pairings

### Em Desenvolvimento 🔄
- Teste automatizado de providers
- Switch automático baseado em carga
- Dashboard web de custos

### Planejado 📅
- Load balancing entre providers
- A/B testing automático
- Cache compartilhado entre providers
- Otimização automática de custos
- Alertas de orçamento
- Análise de qualidade de respostas

---

**Status**: ✅ Implementado e Funcionando
**Build**: v2.2 Multi-Provider Edition
**Date**: 2026-02-09

Aproveite o poder de múltiplos LLMs! 🔌🤖

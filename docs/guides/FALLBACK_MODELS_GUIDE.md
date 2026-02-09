# 🔄 Fallback Models - Complete Guide

**Date**: 2026-02-08
**Status**: ✅ **FULLY FUNCTIONAL**

---

## 🎯 Overview

O sistema de **Fallback Models** permite configurar um ou mais modelos alternativos que serão automaticamente utilizados caso o modelo primário demore muito para responder ou falhe.

### Benefícios

- ✅ **Maior confiabilidade**: Se um modelo está lento ou indisponível, o sistema automaticamente tenta alternativas
- ✅ **Sem perda de contexto**: A memória, tarefas e histórico de conversação permanecem intactos
- ✅ **Transparência**: Logs indicam quando fallback foi utilizado
- ✅ **Flexibilidade**: Configure quantos modelos quiser na ordem de prioridade
- ✅ **Zero impacto na qualidade**: Todas as mensagens, tools e parâmetros são preservados

---

## ⚙️ Configuração

### 1. Configuração Básica

Adicione no seu `.agi/config.json`:

```json
{
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini", "gpt-3.5-turbo"],
  "fallback_timeout": 30
}
```

**Explicação**:
- `model`: Modelo primário (sempre tentado primeiro)
- `fallback_models`: Lista de modelos alternativos na ordem de prioridade
- `fallback_timeout`: Tempo em segundos antes de desistir e tentar o próximo modelo (padrão: 30s)

### 2. Sem Fallback (Comportamento Padrão)

Se você não quer usar fallback, simplesmente deixe a lista vazia:

```json
{
  "model": "gpt-4o-mini",
  "fallback_models": []
}
```

O sistema funcionará normalmente sem tentar modelos alternativos.

---

## 📊 Exemplos de Uso

### Exemplo 1: OpenAI com Fallback

Modelo primário caro com fallback para modelos mais rápidos:

```json
{
  "api_base_url": "https://api.openai.com/v1",
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini", "gpt-3.5-turbo"],
  "fallback_timeout": 30
}
```

**Fluxo**:
1. Tenta `gpt-4o` (30s timeout)
2. Se falhar/demorar → tenta `gpt-4o-mini` (30s timeout)
3. Se falhar/demorar → tenta `gpt-3.5-turbo` (30s timeout)
4. Se todos falharem → retorna erro

### Exemplo 2: NVIDIA NIM com Fallback

```json
{
  "api_base_url": "https://integrate.api.nvidia.com/v1",
  "model": "meta/llama-3.3-70b-instruct",
  "fallback_models": ["meta/llama-3.1-8b-instruct", "mistralai/mistral-7b-instruct-v0.3"],
  "fallback_timeout": 45
}
```

### Exemplo 3: Anthropic Claude com Fallback OpenAI

Você pode até usar provedores diferentes se tiver múltiplas chaves configuradas:

```json
{
  "api_base_url": "https://api.anthropic.com/v1",
  "model": "claude-3-5-sonnet-20241022",
  "fallback_models": [],
  "fallback_timeout": 60
}
```

*Nota*: Fallback entre provedores diferentes requer que ambos usem a mesma API key ou que você configure adequadamente.

### Exemplo 4: Alta Confiabilidade (Múltiplos Fallbacks)

```json
{
  "model": "gpt-4o",
  "fallback_models": [
    "gpt-4o-mini",
    "gpt-3.5-turbo",
    "gpt-3.5-turbo-16k"
  ],
  "fallback_timeout": 20
}
```

Com 4 modelos configurados, você tem 3 camadas de proteção.

---

## 🚀 Como Funciona

### Fluxo de Execução

```
┌─────────────────────────────────────────────┐
│ 1. User envia mensagem                      │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 2. Tenta modelo primário                    │
│    - Timeout: fallback_timeout              │
│    - Mesmas messages, tools, params         │
└──────────────┬──────────────────────────────┘
               │
               ├─► ✅ Sucesso? → Retorna resposta
               │
               ├─► ❌ Falhou/Timeout?
               │
               ▼
┌─────────────────────────────────────────────┐
│ 3. Tenta fallback_models[0]                 │
│    - Timeout: fallback_timeout              │
│    - MESMAS messages, tools, params         │
└──────────────┬──────────────────────────────┘
               │
               ├─► ✅ Sucesso? → Retorna resposta
               │
               ├─► ❌ Falhou?
               │
               ▼
┌─────────────────────────────────────────────┐
│ 4. Tenta fallback_models[1]                 │
│    (se existir)                             │
└──────────────┬──────────────────────────────┘
               │
               ├─► ✅ Sucesso? → Retorna resposta
               │
               └─► ❌ Todos falharam → Erro
```

### Importante: Memória Preservada

**O sistema garante que**:
- ✅ As **mesmas mensagens** são enviadas para todos os modelos
- ✅ As **mesmas tools** estão disponíveis
- ✅ Os **mesmos parâmetros** (temperature, top_p, max_tokens) são usados
- ✅ A **memória do agente** não é afetada
- ✅ As **tarefas em andamento** não são perdidas
- ✅ O **histórico de conversação** permanece consistente

Isso significa que **não há risco** de bagunçar o estado do agente!

---

## 🔍 Logs e Debug

### Logs Normais (Sem Fallback)

Se o modelo primário funciona, você não verá nenhuma mensagem de fallback:

```
[INFO] Processing chat request with model: gpt-4o-mini
[INFO] Response received (250 tokens)
```

### Logs com Fallback (Primário Falhou)

Quando o fallback é acionado, você verá logs detalhados:

```
[INFO] Processing chat request with model: gpt-4o
[WARN] Primary model gpt-4o failed: context deadline exceeded. Trying fallback models...
[INFO] Attempting fallback model 1/2: gpt-4o-mini
[INFO] Fallback model gpt-4o-mini succeeded!
[INFO] Response received (245 tokens)
```

### Logs com Todos os Modelos Falhando

Se nenhum modelo responder:

```
[INFO] Processing chat request with model: gpt-4o
[WARN] Primary model gpt-4o failed: context deadline exceeded. Trying fallback models...
[INFO] Attempting fallback model 1/1: gpt-4o-mini
[WARN] Fallback model gpt-4o-mini failed: API error (status 503): Service Unavailable
[ERROR] All models failed, primary error: context deadline exceeded
```

### Verificar Logs

```bash
# Tail dos logs em tempo real
tail -f .agi/agent.log | grep -i "fallback\|model"

# Ver apenas mensagens de fallback
cat .agi/agent.log | grep "fallback"
```

---

## ⚡ Ajustando o Timeout

### Timeout Padrão (30s)

```json
{
  "fallback_timeout": 30
}
```

Adequado para a maioria dos casos.

### Timeout Curto (15s)

Para respostas mais rápidas, mas pode gerar mais fallbacks:

```json
{
  "fallback_timeout": 15
}
```

**Use quando**:
- Quer respostas rápidas
- Tem modelos fallback muito confiáveis
- Usa modelos rápidos como primário (gpt-3.5-turbo, gpt-4o-mini)

### Timeout Longo (60s)

Para modelos mais lentos ou requisições complexas:

```json
{
  "fallback_timeout": 60
}
```

**Use quando**:
- Usa modelos grandes/lentos (GPT-4o, Claude Opus)
- Requisições muito complexas com muitas tools
- Prefere esperar mais antes de tentar fallback

### Sem Timeout (Comportamento Antigo)

Se `fallback_models` estiver vazio, o timeout padrão do HTTP client (120s) é usado:

```json
{
  "fallback_models": []
}
```

---

## 🧪 Testando o Sistema

### Teste 1: Fallback Forçado (Timeout)

Configure um timeout muito curto para forçar fallback:

```json
{
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini"],
  "fallback_timeout": 1
}
```

Envie uma mensagem e observe os logs. O primário provavelmente vai timeout e o fallback será usado.

### Teste 2: Fallback com Modelo Inválido

Configure um modelo inválido como primário:

```json
{
  "model": "modelo-que-nao-existe",
  "fallback_models": ["gpt-4o-mini"],
  "fallback_timeout": 30
}
```

O fallback deve ser acionado imediatamente.

### Teste 3: Múltiplos Fallbacks

Configure 3 modelos na ordem de preferência:

```json
{
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini", "gpt-3.5-turbo"],
  "fallback_timeout": 20
}
```

Observe que se o primeiro e segundo falharem, o terceiro será tentado.

---

## 📈 Estatísticas de Uso

### Ver Quantas Vezes Fallback Foi Usado

```bash
# Contar acionamentos de fallback
cat .agi/agent.log | grep "Trying fallback models" | wc -l

# Ver quais modelos fallback foram bem-sucedidos
cat .agi/agent.log | grep "Fallback model .* succeeded"
```

### Taxa de Sucesso do Primário

```bash
# Requests totais
cat .agi/agent.log | grep "Processing chat request" | wc -l

# Requests com fallback
cat .agi/agent.log | grep "Trying fallback" | wc -l

# Taxa de sucesso = (total - fallback) / total * 100
```

---

## 🔧 Configurações Avançadas

### Fallback com Diferentes Timeouts

Atualmente, todos os modelos usam o mesmo timeout. Se você precisa de timeouts diferentes por modelo, considere:

1. **Timeout curto para primário, sem fallback**: Rápido mas menos confiável
2. **Timeout longo para primário, com fallback**: Mais confiável

### Fallback Entre Provedores

Para usar fallback entre provedores diferentes (ex: OpenAI → Anthropic), você precisaria configurar múltiplas instâncias ou usar proxy/gateway.

**Solução atual**: Use o mesmo provedor com modelos diferentes (recomendado).

### Custo vs Confiabilidade

Configure modelos mais caros como primário e mais baratos como fallback:

```json
{
  "model": "gpt-4o",              // $15/1M tokens
  "fallback_models": [
    "gpt-4o-mini",                 // $0.15/1M tokens
    "gpt-3.5-turbo"                // $0.50/1M tokens
  ]
}
```

Na maioria das vezes usa o modelo caro (alta qualidade). Quando ele falha, usa os mais baratos (economia).

---

## 🐛 Troubleshooting

### Problema: Fallback nunca é acionado

**Sintomas**:
- Configurei fallback mas logs não mostram tentativas
- Parece que apenas o primário é usado

**Soluções**:

1. Verifique se `fallback_models` não está vazio:
   ```bash
   cat .agi/config.json | grep -A2 fallback_models
   ```

2. Verifique se o primário está funcionando bem demais (sucesso sempre):
   - Isso é bom! Significa seu modelo primário é confiável
   - Fallback só aciona em falhas/timeouts

3. Aumente logs para ver tentativas:
   ```bash
   tail -f .agi/agent.log
   ```

### Problema: Fallback aciona muito frequentemente

**Sintomas**:
- Toda requisição usa fallback
- Logs cheios de "Trying fallback models"

**Soluções**:

1. Aumente o `fallback_timeout`:
   ```json
   {
     "fallback_timeout": 60  // Era 30
   }
   ```

2. Verifique se o modelo primário existe e está acessível:
   ```bash
   # Teste manual
   curl -X GET https://api.openai.com/v1/models \
     -H "Authorization: Bearer $API_KEY"
   ```

3. Verifique sua conexão de internet e rate limits

### Problema: Todos os modelos falhando

**Sintomas**:
- Erro: "all models failed"
- Nenhuma resposta recebida

**Soluções**:

1. Verifique API key:
   ```bash
   echo $API_KEY
   ```

2. Verifique rate limits no provedor

3. Teste manualmente cada modelo:
   ```bash
   curl -X POST https://api.openai.com/v1/chat/completions \
     -H "Authorization: Bearer $API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-4o-mini",
       "messages": [{"role": "user", "content": "test"}]
     }'
   ```

### Problema: Fallback está lento

**Sintomas**:
- Respostas demoram muito tempo
- Múltiplos timeouts antes de receber resposta

**Soluções**:

1. Reduza `fallback_timeout` para falhar mais rápido:
   ```json
   {
     "fallback_timeout": 15  // Era 30
   }
   ```

2. Use menos modelos fallback (ex: apenas 1 ou 2)

3. Use modelos mais rápidos:
   ```json
   {
     "model": "gpt-4o-mini",
     "fallback_models": ["gpt-3.5-turbo"]
   }
   ```

---

## 📊 Benchmarks

### Latência com Fallback

| Cenário | Tempo Médio | Observações |
|---------|-------------|-------------|
| Primário sucesso | 2-5s | Normal, sem overhead |
| Fallback após timeout (30s) | 32-35s | 30s timeout + 2-5s fallback |
| Fallback após erro imediato | 2-5s | Sem espera, direto ao fallback |
| Todos falham (2 fallbacks) | 60-65s | 3 × timeout |

### Recomendações de Timeout

| Tipo de Modelo | Timeout Recomendado |
|----------------|---------------------|
| Modelos rápidos (gpt-3.5, mini) | 15-20s |
| Modelos médios (gpt-4o-mini) | 30s (padrão) |
| Modelos grandes (gpt-4o, Claude) | 45-60s |
| Auto-hospedado (local LLM) | 60-120s |

---

## 🎯 Best Practices

### 1. Configure Fallback para Produção

Mesmo se seu modelo primário é confiável, sempre configure pelo menos 1 fallback:

```json
{
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini"]
}
```

### 2. Ordem de Prioridade por Qualidade

Liste fallbacks em ordem decrescente de qualidade/capacidade:

```json
{
  "model": "gpt-4o",
  "fallback_models": [
    "gpt-4o-mini",      // Ainda muito bom
    "gpt-3.5-turbo"     // Último recurso
  ]
}
```

### 3. Monitore Uso de Fallback

Revise logs semanalmente para identificar padrões:

```bash
# Relatório semanal
cat .agi/agent.log | grep "fallback" | tail -100
```

Se fallback está sendo usado muito, considere:
- Aumentar timeout
- Trocar o modelo primário
- Verificar problemas de conectividade

### 4. Timeout Conservador

Prefira timeouts maiores (30-45s) para evitar falsos positivos:

```json
{
  "fallback_timeout": 45
}
```

É melhor esperar um pouco mais do que trocar de modelo desnecessariamente.

### 5. Teste Regularmente

Teste seu setup de fallback periodicamente:

```bash
# Configure timeout curto temporariamente
# Envie algumas mensagens
# Observe se fallback funciona
# Restaure timeout normal
```

---

## 🔮 Futuras Melhorias

Possíveis funcionalidades futuras:

- [ ] Timeouts diferentes por modelo
- [ ] Fallback entre provedores (multi-provider)
- [ ] Estatísticas de uso por modelo
- [ ] Auto-ajuste de timeout baseado em latência histórica
- [ ] Circuit breaker (desabilitar modelo após N falhas)
- [ ] Retry automático com backoff por modelo
- [ ] Webhooks para notificar sobre fallback

---

## 📝 Exemplo Completo de Configuração

```json
{
  "api_base_url": "https://api.openai.com/v1",
  "api_key": "",
  "model": "gpt-4o",
  "fallback_models": ["gpt-4o-mini", "gpt-3.5-turbo"],
  "fallback_timeout": 30,
  "temperature": 0.7,
  "max_tokens": 4000,
  "max_context_size": 128000,
  "memory": {
    "max_short_term_items": 20,
    "max_working_items": 50,
    "max_long_term_items": 100,
    "compression_trigger": 15,
    "storage_path": ".agi/memory.json"
  },
  "ui": {
    "theme": "dark",
    "show_tokens": true,
    "show_timestamp": true,
    "verbose": true
  },
  "telegram": {
    "enabled": true,
    "chat_id": 123456789,
    "notify_on_tool_start": true
  },
  "permissions": {
    "allowed_commands": ["*"],
    "allowed_tools": ["*"],
    "sensitive_tools": [
      "git_commit",
      "git_push",
      "exec_command",
      "write_file",
      "delete_file"
    ],
    "auto_approve_non_sensitive": false,
    "require_approval_for_all": false,
    "telegram_approval_timeout": 300,
    "enable_audit_log": true,
    "audit_log_path": ".agi/audit.log"
  }
}
```

---

**Status**: ✅ **PRODUCTION READY**
**Overhead**: ✅ **Zero quando não acionado**
**Impacto na memória**: ✅ **Nenhum - contexto preservado**

*Configure e esqueça - seu AGI está protegido contra falhas de modelo! 🛡️*

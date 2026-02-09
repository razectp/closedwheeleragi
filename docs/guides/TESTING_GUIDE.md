# 🧪 Guia de Testes - Correções Implementadas

## ✅ Build Completo

```bash
✅ Binary: ClosedWheeler.exe
✅ Size: 13MB
✅ Compile Errors: 0
✅ Date: 2026-02-09 16:56
```

---

## 🎯 Testes a Realizar

### Teste 1: Verificar Banner (Problema 2)

**Objetivo:** Confirmar que banner não tem caracteres malformados

**Passos:**
```bash
./ClosedWheeler.exe
```

**Resultado Esperado:**
```
  ╔═══════════════════════════════════════════════════════════════╗
  ║                                                               ║
  ║          ClosedWheelerAGI - Intelligent Coding Agent          ║
  ║                                                               ║
  ║                        Version 0.1.0                          ║
  ║                                                               ║
  ╚═══════════════════════════════════════════════════════════════╝
```

**✅ SUCESSO SE:**
- Banner aparece limpo
- Sem caracteres estranhos ou backticks
- Box-drawing characters corretos

**❌ FALHA SE:**
- Caracteres malformados
- Boxes quebrados
- Encoding issues

---

### Teste 2: Verificar workplace (Problema 1 - CRÍTICO)

**Objetivo:** Garantir que workplace/workplace NUNCA é criado

**Passos:**
```bash
# Cenário 1: Execução normal
cd C:\Users\cezar\OneDrive\Área de Trabalho\ClosedWheelerAGI
./ClosedWheeler.exe

# Verificar estrutura
ls -la workplace/
# Deve mostrar APENAS: .agirules, personality.md, expertise.md
# NÃO deve mostrar: workplace/ (subdiretório)

# Cenário 2: Execução de dentro de workplace (TESTE CRÍTICO)
cd workplace
../ClosedWheeler.exe

# Verificar estrutura
cd ..
ls -la workplace/
# Ainda deve mostrar APENAS arquivos, SEM workplace/ aninhado
```

**✅ SUCESSO SE:**
```
workplace/
├── .agirules
├── personality.md
└── expertise.md

[SEM workplace/workplace/]
```

**❌ FALHA SE:**
```
workplace/
├── workplace/           ← ISTO NÃO DEVE EXISTIR
│   └── ...
```

---

### Teste 3: Multi-Window System (Problema 3 - NOVO)

**Objetivo:** Testar sistema de janelas separadas para cada agente

#### 3A. Iniciar Dual Session

**Passos:**
```bash
./ClosedWheeler.exe

# No TUI:
/session on
```

**Resultado Esperado:**
```
✅ Dual session enabled
Configure agents with /agents command
```

#### 3B. Iniciar Debate (Background)

**Passos:**
```bash
/debate "Should AI have rights?" 10
```

**Resultado Esperado:**
```
🤖 Starting debate on: Should AI have rights?
Max turns: 10

💡 Tip: Use /conversation to open separate windows for each agent!
   🔵 Window 1 = Agent A only
   🟢 Window 2 = Agent B only

   The debate will run in the background while you continue working.
```

**✅ SUCESSO SE:**
- Mensagem aparece no TUI
- Nenhuma janela abre automaticamente
- Debate começa em background

**❌ FALHA SE:**
- Janelas abrem automaticamente
- Erro ao iniciar debate

#### 3C. Abrir Multi-Window

**Passos:**
```bash
/conversation
```

**Resultado Esperado:**
```
1. DUAS JANELAS PowerShell DEVEM ABRIR:

   [JANELA 1 - PowerShell]
   Título: "Agent A (Blue)"

   ╔══════════════════════════════════════════════════════╗
   ║              🔵  Agent A  WINDOW  🔵                ║
   ╚══════════════════════════════════════════════════════╝

   📺 This window shows only Agent A messages

   ⏳ Waiting for debate to start...
   ═══════════════════════════════════════════════════════

   Turn 1 - 15:30:45
   ────────────────────────────────────────────────────────
   [Agent A's message here]


   [JANELA 2 - PowerShell]
   Título: "Agent B (Green)"

   ╔══════════════════════════════════════════════════════╗
   ║              🟢  Agent B  WINDOW  🟢                ║
   ╚══════════════════════════════════════════════════════╝

   📺 This window shows only Agent B messages

   ⏳ Waiting for debate to start...
   ═══════════════════════════════════════════════════════

   Turn 2 - 15:31:02
   ────────────────────────────────────────────────────────
   [Agent B's message here]

2. TUI PRINCIPAL MOSTRA:
   ✅ Agent windows opened!

   📺 The debate is now shown in TWO separate terminal windows:

   🔵 Window 1: Agent A only
   🟢 Window 2: Agent B only

   You can continue working here while watching the debate in real-time.
```

**✅ SUCESSO SE:**
- 2 janelas PowerShell abrem
- Janela 1 mostra APENAS Agent A
- Janela 2 mostra APENAS Agent B
- Mensagens aparecem em tempo real
- Headers corretos (🔵 e 🟢)
- TUI principal permanece funcional

**❌ FALHA SE:**
- Janelas não abrem
- Apenas 1 janela abre
- Mensagens misturadas nas janelas
- Janelas não atualizam em tempo real
- Erro no TUI

#### 3D. Parar Debate

**Passos:**
```bash
/stop
```

**Resultado Esperado:**
```
1. TUI PRINCIPAL:
   ⏹️ Conversation Stopped

   The debate has been ended early.

   Final Statistics:
   - Total messages: X
   - Agent A: Y messages
   - Agent B: Z messages

2. JANELAS POWERSHELL:
   ═══════════════════════════════════════════════════════
   🏁 Debate Ended
   You can close this window now.
   ═══════════════════════════════════════════════════════

   [Janelas permanecem abertas para revisão]
```

**✅ SUCESSO SE:**
- Debate para
- Estatísticas aparecem no TUI
- Janelas mostram mensagem de fim
- Janelas permanecem abertas
- Usuário pode fechar manualmente

---

## 🔄 Teste de Fallback

**Objetivo:** Verificar fallback automático se janelas falharem

**Passos:**
```bash
# Simular falha (ex: PowerShell não disponível - difícil de testar)
/conversation
```

**Resultado Esperado (se falhar):**
```
❌ Failed to open agent windows: [error message]

Falling back to TUI view...

[Mensagens do debate aparecem no TUI principal]
🔵 Agent A (Turn 1)
────────────────────────────────────────────────────────────
[message]

🟢 Agent B (Turn 2)
────────────────────────────────────────────────────────────
[message]
```

**✅ SUCESSO SE:**
- Erro é mostrado claramente
- Sistema volta para TUI view automaticamente
- Debate continua visível no TUI
- Sem crash

---

## 📁 Verificação de Arquivos

**Objetivo:** Confirmar estrutura de arquivos correta

**Verificar:**
```bash
ls -la .agi/
```

**Resultado Esperado:**
```
.agi/
├── config.json
├── conversation_live.txt  [pode existir ainda]
├── agent_a.txt           [NOVO - criado ao usar /conversation]
├── agent_b.txt           [NOVO - criado ao usar /conversation]
└── debug.log
```

**Verificar:**
```bash
ls -la workplace/
```

**Resultado Esperado:**
```
workplace/
├── .agirules
├── personality.md
├── expertise.md
└── [arquivos do usuário, se houver]

[SEM workplace/ aninhado]
```

---

## 🐛 Se Encontrar Bugs

### Debug Steps:

1. **Verificar logs:**
```bash
tail -f .agi/debug.log
```

2. **Testar comando PowerShell manualmente:**
```powershell
# Windows
$host.ui.RawUI.WindowTitle='Test'; Get-Content '.agi/agent_a.txt' -Wait -Tail 100
```

3. **Verificar permissões:**
```bash
ls -la .agi/
ls -la workplace/
```

4. **Reportar:**
- Qual teste falhou
- Mensagem de erro exata
- Screenshot se possível
- Conteúdo de `.agi/debug.log`

---

## ✅ Checklist Completo

### Teste 1: Banner
- [ ] Banner aparece limpo
- [ ] Sem caracteres malformados
- [ ] Box-drawing correto

### Teste 2: workplace
- [ ] Execução normal: workplace/ criado corretamente
- [ ] Execução de dentro de workplace/: SEM workplace/workplace/
- [ ] Arquivos do usuário preservados
- [ ] Apenas .agirules, personality.md, expertise.md criados

### Teste 3: Multi-Window
- [ ] /session on funciona
- [ ] /debate inicia em background
- [ ] Nenhuma janela abre automaticamente
- [ ] /conversation abre 2 janelas PowerShell
- [ ] Janela 1 mostra APENAS Agent A
- [ ] Janela 2 mostra APENAS Agent B
- [ ] Headers corretos (🔵 e 🟢)
- [ ] Mensagens em tempo real
- [ ] TUI principal continua funcional
- [ ] /stop para o debate
- [ ] Janelas mostram mensagem de fim
- [ ] Estatísticas corretas no TUI

### Teste 4: Fallback
- [ ] Se janelas falharem, TUI view ativa automaticamente
- [ ] Debate continua visível
- [ ] Sem crash

### Teste 5: Arquivos
- [ ] .agi/agent_a.txt criado
- [ ] .agi/agent_b.txt criado
- [ ] workplace/ estrutura correta
- [ ] SEM workplace/workplace/

---

## 🎉 Critérios de Aceitação

**TODOS os testes devem passar para considerar SUCESSO COMPLETO:**

1. ✅ Banner limpo
2. ✅ workplace/ correto (SEM duplicação)
3. ✅ Multi-window abre 2 janelas separadas
4. ✅ Mensagens aparecem nas janelas corretas
5. ✅ Real-time updates funcionam
6. ✅ Fallback automático funciona se necessário

**Se QUALQUER teste falhar:**
- Documentar o erro
- Verificar logs
- Reportar para correção

---

**Pronto para testar!** 🚀

# 02 — Matriz de responsabilidade (Linear ↔ Orca sem redundância)

Princípio: **cada fato tem um dono único.** Redundância de responsabilidade é a raiz
de drift, painéis que mentem e trabalho duplicado.

| Fonte | É dona de | Nunca é |
|---|---|---|
| **Linear** | Existência da task, spec (na description — endereço único), prioridade, projeto, estado cru→pronto (Backlog/Todo/In Progress/Done), memória durável (comentários de interview, `## Lessons`, `## Outcome`), trailers `repo:` e `agent-config:` | UI de execução |
| **Orca** | Worktrees (junção com o plano via `linkedLinearIssue`), terminais, estado vivo do agente (hooks), canal de decisão (gates/mensagens), pulso (automations) | Tracker |
| **GitHub** | PRs, merge state, CI; fila dev→main sempre **derivada** de `gh`/git | Armazém de fila |
| **dotfiles / management-hub** | Política versionada: scripts/código, rubrica de interview, prompt-contrato do agente, tabela de alocação modelo/effort | Depósito de dados (ledger → `~/.local/state/work-os/`) |
| **Hub** | Projeção + despacho de comandos que agem na fonte | Fonte de verdade. Exceção declarada: snapshot/cache **derivado e apagável** |

## Escritores de estado do Linear (um por transição)

| Transição | Escritor único | Condição |
|---|---|---|
| (captura) → Backlog | `work` | sempre pode; nunca falha por projeto ausente |
| Backlog → Todo | `triage` | **só se a spec foi persistida na description** (senão: não promove, vira pendência "spec não persistida") |
| Todo → In Progress | `linear-sync` | **só após agente confirmado** (`terminal wait --for tui-idle` + verificação pós-send) — In Progress reflete fato consumado |
| → Done | `ship` | exige PR MERGED via `gh`; remove worktree; appenda ledger |
| Reconciliação de drift | `health-sweep` (automation) | ver tabela abaixo |

**Edição humana no Linear é evento legítimo.** Todo escritor é idempotente e
convergente sob ela — nunca assume que o estado atual foi escrito por ele.

## O reconciliador (health-sweep)

Drift é inevitável entre três sistemas; o que é proibido é drift **sem dono**.
A automation health-sweep é o quinto escritor declarado, com tabela drift→ação:

| Drift detectado | Ação |
|---|---|
| In Progress sem worktree ativo/agente vivo | volta a Todo (ou alerta blocking se há PR aberto) |
| PR merged + worktree vivo ("falta ship") | pendência acionável no hub / dispara `ship` |
| Todo com gates de interview pendentes | rebaixa a Backlog |
| Worktree sem `linkedLinearIssue` (órfão) | pendência hygiene |

## Regras estruturais

1. **`gate-resolve` transacional único**: (1) escreve o comentário estruturado na
   issue Linear; (2) só então resolve o gate no Orca. Falhou o Linear → gate vive.
   Triage e hub chamam o mesmo caminho. MAX_ROUNDS conta marcador estruturado
   escrito exclusivamente por ele.
2. **Mensagens Orca são wake-up, nunca estado**: tudo que o hub exibe é rederivável
   de Linear/GitHub; `worker_done`/`merge_ready` servem de trigger de refresh e são
   descartáveis sem perda. O watcher é o único consumidor do inbox; painéis peekam.
3. **Um único predicado de slot**: "worktree ativo com `linkedLinearIssue` e agente
   vivo" — usado por dedup, cap e badge de capacidade. (Historicamente eram dois
   predicados divergentes e o slot vazava.)
4. **Junção issue↔worktree é 1:N no tempo, 1:1 no presente**: invariante "no máximo
   um worktree ativo por issue"; dedup/zumbi/badge avaliam o worktree ativo
   corrente, não "existe worktree".
5. **Decompose estaciona a mãe**: issue com sub-issues não-Done nunca é spawnável.
6. **Fila computada num lugar só** (`linear-sync --queue --json` → depois core
   deste repo); painéis apenas renderizam. Ordenação: prioridade Linear
   (urgent→low, sem-prioridade por último), depois número da key.
7. **Trailer `agent-config` é o único registro durável de modelo/effort**: picker e
   escalação escrevem o trailer via API **antes** de spawnar/respawnar; a tabela de
   alocação gera apenas o default; env é debug explícito. Toda escalação deixa
   rastro na issue.

# 06 — Avaliação: ferramentas de sessões/orquestração de agentes (jul/2026)

Pergunta: "já existem ferramentas tmux que gerenciam sessões de agentes — usar uma
delas em vez de construir backend próprio (HEN-83)? E orquestração/paralelismo?"

Método: 4 trilhos de pesquisa paralelos (ecossistema geral, orquestração, deep-dive
em dmux/claude-squad/workmux/amux, mineração comunitária HN/awesome-lists — Reddit
inacessível a crawler, registrado como lacuna). ~40 ferramentas mapeadas, 10 com
leitura de código-fonte. Critérios: spawn scriptável, worktree com base `dev`,
prompt+modelo/effort por spawn, telemetria legível, attach tmux real, **backend
componível** (fila/triagem/UI são nossas), saúde do projeto, orquestração real.

## Veredito

**Adotar [workmux](https://github.com/raine/workmux) como backend de
sessões+worktrees** (HEN-83 re-escopada de "construir spike próprio" para "validar
workmux"). Complemento: o padrão da camada CLI do
[amux/mixpeek](https://github.com/mixpeek/amux) para repasse de `--model/--effort`.
Nenhum runtime de orquestração adotado — ver seção Orquestração.

## Finalistas

| Ferramenta | Veredito | Evidência-chave |
|---|---|---|
| **workmux** (~1.9k★, Rust, 0.x) | **Adotar como backend** | `workmux add <branch> --base dev -P spec.md -b` = o spawn do linear-sync em 1 comando; `list/status --json`, `wait wt1 wt2 --status done` (join), `--foreach` (fan-out templatizado), `--max-concurrent` (cap), `send`, hooks com env vars; "git-worktree-centric, not task-centric" — zero fila/UI imposta; telemetria pela MESMA mecânica de hooks do Claude Code que já usamos |
| **amux** CLI (mixpeek, ~300★) | Adoção parcial / padrão | Único que repassa `--model/--effort/--max-budget-usd` por sessão via CLI; mas worktrees só via servidor (sem base branch), telemetria por scraping, kanban próprio duplicaria o Linear |
| agent-deck (~530★) | Reserva | CLI real + `--json` + tmux nativo; sem base branch configurável, modelo via wrapper `--cmd` |

## Descartados (motivo em uma linha)

- **dmux** — cockpit TUI; spawn só interativo (API REST removida do main), sem `--model` por task, merge próprio com auto-commit IA, dependência OpenRouter.
- **claude-squad** (8.1k★) — spawn headless não existe (issue #157), base branch é TODO no código, zero orquestração, AGPL.
- **ccmanager** — substitui tmux por PTYs próprias (anti-requisito nº1); design dos status-hooks vale como inspiração.
- **ntm** — scriptável e tmux nativo, mas impõe fila/triagem próprias (beads) competindo com o Linear; swarm-num-repo, autor único sem PRs.
- **Peso-pesados 2026** (symphony 26k★, cmux 24.7k★, herdr 17.9k★, Gas Town 17.1k★, orca 21.7k★): todos são *produtos/ADEs* que substituiriam o work-os inteiro, não motores. Gas Town é o único com orquestração completa (DAG, mailboxes, merge train) — e custa ~$100/h, troca o Linear por Beads e o gargalo vira alimentar a máquina.
- Mortalidade do mercado como lição: vibe-kanban (27k★) sunset, Crystal deprecated, Omnara arquivado — **integração fina e substituível, nunca fundir o sistema na ferramenta.**

## Orquestração e paralelismo (achados)

1. Nenhuma ferramenta tmux-nativa tem DAG/dependências reais — "2025 era N sessões
   em abas" continua majoritariamente verdade. workmux é o único com primitivas
   imperativas úteis (fan-out/join/cap) que compõem com pipeline próprio.
2. **Claude Code nativo virou o orquestrador**: agent view/background sessions
   (`claude --bg`, `claude agents --json`, worktrees automáticas, supervisor) +
   Agent Teams experimental (task deps, mailbox, plan approval, hooks de quality
   gate exit-2). Cobre ~90% das primitivas de orquestração de que precisamos —
   fan-out intra-task e pipelines spec→implement→review vêm daí, não de runtime
   externo. Risco: preview/experimental, churn entre releases.
3. Prática 2026 (HN, Osmani, DORA): o gargalo é **verificação, não geração**; teto
   humano de 2–3 sessões paralelas; o padrão vencedor é exatamente o nosso — fila
   de issues → worktrees independentes → review no final. DAG persistente e
   mensagens inter-agente: custo > benefício para solo.
4. MS Conductor (YAML determinístico, approval gates) é a única peça de pipeline
   adotável isoladamente no futuro, por baixo da fila Linear.

## Integração workmux (decisões)

- linear-sync chama `workmux add` no lugar de `orca worktree create` + poll +
  `terminal send` (~80 linhas morrem). Orca sai do critical path; app fica como
  utilitário opcional.
- Modelo/effort: perfis pré-declarados no config do workmux (`claude-opus-high`,
  `claude-sonnet-medium`, `claude-haiku-low`…) **gerados a partir da tabela de
  política** (a política já é 3–4 classes — perfis casam melhor que flags livres).
- Telemetria: nossos hooks JSONL (HEN-82) continuam a fonte primária; `workmux
  status --json` é reconciliação. Não instalar os hooks do workmux por cima dos
  nossos sem revisar sobreposição.
- `workmux merge` fica FORA do fluxo — `ship` continua dono do fechamento
  (PR → review dev→main).
- Risco bus-factor-1/churn 0.x: pinar versão; a integração é fina (CLI+git+tmux),
  substituível por ~150 linhas próprias se o projeto morrer (plano B validado
  pela própria avaliação).

## Condições de revisão

- Se workmux ganhar flags livres de agente por spawn, simplificar perfis.
- Se Agent Teams/agent view estabilizarem (sair de experimental), reavaliar quanto
  do workmux o Claude Code nativo absorve (worktrees automáticas + supervisor).
- Se surgir necessidade real de pipeline declarativo: MS Conductor por baixo da
  fila, não Gas Town.

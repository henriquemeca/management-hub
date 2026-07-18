# 03 — O hub (painel único de atenção e comando)

Requisitos: criar tasks; ver as tasks de todos os projetos com prioridade e estado
(crua → pronta); ver sessões abertas — se precisam de mim ou de review; iterar a
estrutura, as skills e os processos. Tudo numa tela, com ação de uma tecla.

## Arquitetura: watcher escreve, painel lê

Projeção stateless de 4 fontes de rede não abre em <100ms — e painel lento não vira
reflexo. Solução:

- **Watcher** (loop de ~15s + triggers de evento via `orca orchestration check
  --peek`): consolida Linear + `orca worktree ps`/gates/inbox + `gh`/git num
  `snapshot.json` (ex.: `~/.cache/wos/snapshot.json`).
- **Painel** (`prefix+A` popup tmux / CLI `wos now`): renderiza do disco
  instantaneamente, dispara refresh assíncrono.
- **Status bar tmux**: lê o mesmo arquivo — zero rede no redraw.
- O snapshot é **cache derivado e apagável**: `rm` só pode custar uma notificação
  repetida. Guarda também a marca d'água de apresentação (timestamp da última
  abertura + itens exibidos) — cache de apresentação, não estado de domínio.

## A tela

**Abre em AGORA** — só interrupções, ordenadas por urgência:

| Tier | Itens | Linha carrega |
|---|---|---|
| `blocking` | gate de interview (pergunta visível na linha), ask explícito de agente | pergunta truncada, issue, idade |
| `ready` | PR dev→main **acima do limiar** (≥N commits ou ≥X dias), falta-ship | nº commits, idade, repo |
| `capacity` | slot livre + próxima da fila | key, prioridade, `agent-config` do trailer |
| `hygiene` | zumbi (vinculado sem agente vivo), órfão, drift, spec não persistida | tipo + ação sugerida |

**Tab → PLANO** (visão sob demanda, filtrada): em-interview (aguardando VOCÊ),
pronta #N-na-fila, rodando (estado+idade do agente), falta-ship, top-5 do Backlog
por prioridade. Toggle para lista completa. O inventário integral de cruas mora no
Linear — o painel projeta o que pode virar ação nesta semana.

**META** (fora da v1; volta quando o ledger existir, reduzida a 2 linhas
acionáveis): data do último retro + tecla para rodar; issues do work-os no pipeline.

## Teclas

- **enter** — ação primária do tipo do item: gate→responder inline (via
  `gate-resolve` transacional), capacity→spawn com config do trailer
  pré-selecionada, PR→abrir, zumbi→limpar, rodando→focar pane.
- **tab** — alterna AGORA/PLANO.
- **ctrl-n** — captura (`work`), único chord global.
- Demais ações (escalar modelo, `triage --only`, spawn com picker alternativo) em
  **menu contextual** do item selecionado (fzf aninhado).
- Proibido: ctrl-s (XOFF congela o terminal), ctrl-r (conflita com history-search).

## Push honesto

- **Notificação macOS + lado `!` do contador**: SÓ gate e escalation — eventos
  estruturalmente decisórios. `waiting` de agente é ambíguo por construção
  ("terminou o turno" vs "perguntou em texto livre") e fica como linha informativa
  com idade, sem push, até o contrato do worker exigir `ask` explícito.
- **Contador = "novo desde a última olhada"** (marca d'água). Pendência crônica
  (dev→main acumulando) aparece no painel mas não re-infla o contador — senão em
  uma semana o dono aprende que `!1` não significa nada e o push morre.
- Debounce por item; nunca notificar em `Stop`/`PostToolUse`.

## Produtores (cobertura completa das filas)

Regra: **tudo que precisa do humano vira gate/mensagem no Orca, ou é derivável de
uma query que o watcher faz.** Produtor que só imprime no stdout quebra o contrato.
Pendências do `linear-sync` (sem trailer repo, repo não registrado, agente que não
subiu) viram `escalation` idempotente por subject.

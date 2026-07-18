# 01 — Visão

## Objetivo

Via terminal, um workflow maduro que permita: **receber ideias cruas, refiná-las,
criar cargas de trabalho paralelas com esforço e modelos ajustados automaticamente,
e colher aprendizados para iterar o próprio sistema.**

O humano é o gargalo de atenção do sistema. O desenho inverte o fluxo: em vez de
"rodar pelo escritório" (abrir worktrees e sessões uma a uma para descobrir quem
precisa de você), **todos vêm à sua mesa** — os agentes já empurram estado via hooks
para o runtime do Orca; o hub agrega as filas de decisão numa tela só, com push
honesto e ação de uma tecla.

## O ciclo

```
captura (work) → triagem (interview assíncrono, Backlog→Todo)
  → fila + spawn paralelo (worktrees Orca, modelo/effort por task)
  → atenção (hub: gates, reviews, capacidade)
  → integração em lote (squash por task na dev; review humana no PR dev→main)
  → ship (Done + limpeza) → ledger + lessons → retro → issues de melhoria
  → (o próprio ciclo processa as melhorias)
```

## Estágios de evolução

| Estágio | Onde | O quê |
|---|---|---|
| v1 — hub fzf | dotfiles | `attn`+`wos` fundidos sobre `snapshot.json`; valida a UX barato |
| v2 — CLI `wos` | este repo | motor graduado (parsers, fila, triagem, spawn) em Go (cobra) com testes; `--json` em tudo; spawn delegado ao workmux; cron/automations chamam a CLI |
| v3 — TUI | este repo | bubbletea sobre o mesmo core/feed, refresh vivo (watcher como trigger) |

**Tese central: o TUI é um renderizador; o produto é o core + o feed.** O padrão é o
mesmo de k9s, lazygit e gh-dash — projeções TUI sobre APIs que existem sem elas.
Começar pelo TUI é o caminho clássico para a lógica migrar para dentro da casca e
nunca mais sair.

## Fronteiras (o que este projeto NÃO é)

- **Não é um tracker.** Linear é a fonte do plano e a UI de edição dele.
- **Não é um multiplexador/IDE.** Orca é dono de worktrees, terminais e agentes;
  o hub foca panes (`orca terminal switch`) e despacha comandos.
- **Não é um segundo GitHub.** O hub carrega o ponteiro ("dev do repo X tem 7
  commits há 3 dias"), nunca o diff — review profunda acontece no PR/worktree.
- **Não guarda estado de domínio.** Projeção + despacho; a única memória é cache
  derivado e apagável.

## Teste de aceite do hub

Passar um dia de trabalho sem abrir o board do Linear nem a lista de PRs —
só com o painel (`prefix+A`) e as notificações. Cada fila que ficar de fora obriga
a "ronda pelo escritório" a continuar, e o hub vira só mais um lugar para olhar.

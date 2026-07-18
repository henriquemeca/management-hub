# management-hub

Hub central do **work-os** — um ciclo de desenvolvimento terminal-first com agentes:
ideias cruas → refinamento → execução paralela → aprendizado → iteração do sistema.

Integra as ferramentas existentes **por projeção e despacho**, nunca por duplicação:

| Papel | Ferramenta |
|---|---|
| Plano (tasks, prioridade, specs, estados) | Linear (team HEN) |
| Execução (worktrees, terminais, agentes, gates) | Orca |
| Integração (PRs, merges, CI) | GitHub |
| Agentes de código | Claude Code (default), pi (alternativo) |
| Política versionada (rubrica, contratos, alocação) | dotfiles / este repo |

## Layout

- **`bin/`** — a implementação v1 (scripts zsh migrados dos dotfiles): `work`
  (captura), `triage` (Backlog→Todo via interview assíncrono), `linear-sync`
  (fila→spawn), `wos` (cockpit), `attn` (painel de atenção), `ship` (fechamento).
  Instalados com `make link` (symlinks em `~/.local/bin` — os caminhos que
  tmux/nvim/automations já usam). O dotfiles mantém só os bindings (tmux
  `prefix+W`/`prefix+A`, `:Work` no nvim).
- **`src/wos/`** — o core Python: os scripts graduam para cá com testes.
- **`docs/`** — os conceitos e contratos do sistema.

## Estágios

1. **Hub v1 (fzf)** — fusão de `attn`+`wos` em `bin/` sobre um `snapshot.json`;
   valida o modelo de interação pelo preço de um script.
2. **CLI `wos`** — motor graduado (parsers, fila, triagem, spawn) como biblioteca
   Python testada + CLI com `--json` em tudo. A CLI é a superfície de automação:
   as automations do Orca chamam ela, o painel renderiza ela.
3. **TUI (Textual)** — casca sobre o mesmo core e o mesmo feed, refresh vivo.
   O TUI é um renderizador; **o produto é o core + o feed**.

## Conceitos

Toda a arquitetura está em `docs/` — leia na ordem:

- [`docs/01-visao.md`](docs/01-visao.md) — objetivo, estágios e fronteiras
- [`docs/02-matriz-responsabilidade.md`](docs/02-matriz-responsabilidade.md) — dono único por fato, escritores, reconciliador
- [`docs/03-hub.md`](docs/03-hub.md) — a tela (AGORA/PLANO/META), snapshot, push honesto
- [`docs/04-meta-loop.md`](docs/04-meta-loop.md) — ledger, lessons, retro, guard-rails de política
- [`docs/05-roadmap.md`](docs/05-roadmap.md) — sequência, critérios de graduação dotfiles→aqui

## Status

Fase de fundação: conceitos e contratos definidos; o código migra dos dotfiles por
graduação (ver roadmap), não por big-bang. Tracking no projeto Linear
[`management-hub`](https://linear.app/henrique-personal/project/management-hub-94f864b3111f).

## Dev

```bash
uv sync          # instala deps (Python 3.12+)
uv run pytest    # testes
uv run ruff check && uv run ruff format --check
```

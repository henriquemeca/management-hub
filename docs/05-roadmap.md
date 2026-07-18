# 05 — Roadmap e graduação

## Fase 0 — fundação (este commit)

- Repo criado com conceitos (`docs/`), CLI Go (padrão kan: `main.go` +
  `commands/`, decisão revisada na HEN-52) e os scripts v1 migrados dos
  dotfiles para `bin/` (instalados via `make link` → `~/.local/bin`, preservando os
  caminhos que tmux/nvim já usam). Dotfiles volta a ser só configuração de máquina.
- Projeto Linear `management-hub` (team HEN) criado; melhorias tracked lá e
  processadas pelo próprio ciclo (dogfood).
- Pendências de infra: criar remote GitHub (`henriquemeca/management-hub`, branches
  main+dev) e registrar no Orca (`orca repo add`) para o spawn rotear issues deste
  repo.

## Fase 1 — AGORA: destravar o que existe (nos scripts de `bin/`)

Ordem por alavancagem; detalhes nas issues do projeto Linear:

1. **Fechar o ciclo de execução**: `ship` integrado ao fluxo; `terminal wait
   --for tui-idle` + verificação pós-send (In Progress só após agente confirmado);
   predicado único de slot (dedup = cap = badge). Mata os 3 vazamentos do cap.
2. **Integridade Linear↔Orca**: `gate-resolve` transacional (comentário no Linear
   primeiro, gate depois); promoção a Todo condicionada à spec na description.
3. **Hub v1**: fundir `attn`+`wos` — AGORA/tab-PLANO, watcher + `snapshot.json`
   (<100ms), contador com marca d'água, push só gate/escalation, enter+tab+ctrl-n.
4. **Fila num lugar só**: `linear-sync --queue --json`; painéis renderizam.
5. **Captura à prova de falha**: fallback offline (`work --drain`) + parse de
   stdout separado de stderr.

## Fase 2 — PRÓXIMO CICLO: fechar os loops

6. **Alocação automática v1**: triage escreve `agent-config` no veredito ready
   (+ validar `xhigh` no linear-sync); escalação manual de um toque com rastro.
7. **Telemetria**: ledger event-log em `~/.local/state/work-os/` + lessons com 3
   perguntas validadas no ship + `policy_version`.
8. **Guard-rails de política**: escalate forçado para issues de política + smoke
   test como gate de PR.
9. **Pulso**: automations do Orca — `triage` a cada 30–60min + health-sweep
   (reconciliador). `linear-sync` fica manual até a visibilidade de custo existir.
10. **Serializar a integração em lote**: branch protection na dev (1 status check)
    + `gh pr merge --squash --auto`.
11. **Graduar o motor**: parsers de trailer, ordenação da fila, rubrica/recálculo e
    máquina de rodadas do triage → comandos Go em `commands/` com testes (primeiro: regressão
    HEN-37). Wrappers zsh finos mantêm a UX; `bin/` encolhe até ser só cola.

## Fase 3 — DEPOIS

12. **CLI `wos` unificada** com `--json` em tudo — o contrato congela; automations
    e painéis passam a consumir a CLI.
13. **`retro`** manual (1 proposta por vez, marco no ledger).
14. **TUI (bubbletea)** sobre o mesmo core/feed, refresh vivo. Última milha sobre um
    feed estável — nunca a fundação.
15. **Upstream Orca**: quando o launcher nativo expuser `--model/--effort`, o
    workaround de `terminal send` (~80 linhas do linear-sync) morre.

## Critérios de graduação script→core (os gatilhos que já dispararam)

Um componente migra de `bin/` (zsh) para `commands/` (Go testado) quando:
contrato compartilhado ganharia 2ª cópia ou já causou bug em dogfood (✔ ordenação
duplicada, ✔ parser HEN-37); há hesitação em mudar lógica por não poder testá-la
(✔ recalc/downgrade do triage); vai rodar agendado/daemon (✔ triage em automation);
ou segunda máquina/usuário.

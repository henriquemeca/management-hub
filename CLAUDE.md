# management-hub — instruções do projeto

Hub central do work-os pessoal do Henrique: core + CLI (`wos`) + futuramente TUI que
projeta e despacha sobre Linear (plano), Orca (execução) e GitHub (integração).
Os conceitos completos vivem em `docs/` — **leia `docs/02-matriz-responsabilidade.md`
antes de mexer em qualquer lógica de estado**.

## Princípios inegociáveis

1. **Cada fato tem um dono único.** Linear = plano; Orca = execução; GitHub = integração;
   este repo/dotfiles = política versionada. O hub não é fonte de verdade de nada.
2. **Um escritor por transição de estado no Linear** (triage: Backlog→Todo;
   linear-sync: Todo→In Progress; ship: →Done; health-sweep: reconciliação de drift).
   Edição humana no Linear é evento legítimo — todo escritor é idempotente/convergente
   sob ela, nunca assume que o estado atual foi escrito por ele.
3. **Mensagens/gates do Orca são canal (wake-up), nunca estado.** Tudo que o hub exibe
   tem que ser rederivável de Linear/GitHub. Gate resolvido → comentário no Linear
   primeiro, gate morre depois (transacional).
4. **Hub stateless com uma exceção declarada**: snapshot/cache derivado e apagável
   (`rm` no cache só pode custar uma notificação repetida — regra testável).
5. **Determinismo → código; julgamento → LLM.** O LLM propõe (veredito, spec,
   classificação), o código decide (recálculo local, downgrade, validação).
6. **Política nunca entra no spawn automático.** Issues que alteram rubrica,
   prompt-contrato ou tabela de alocação são sempre escalate na triagem; o merge de
   política é sempre humano.
7. **O TUI não absorve terminais nem o board.** Orca é dono de worktrees/terminais/
   agentes (o hub foca panes e despacha); Linear é a UI de edição do plano.

## Tooling

- **Python 3.12+**, gerenciado com **uv** (`uv sync`, `uv run`).
- Layout `src/` — pacote `wos`, CLI via entry point `wos`.
- **pytest** para testes (todo código do core nasce com teste — o motivo da graduação
  dos scripts zsh foi exatamente ser intestável lá).
- **ruff** para lint + format.
- TUI (estágio 3): **Textual**. Não adicionar antes do feed JSON estabilizar.
- Contratos JSON (feed do snapshot, ledger, trailers) são API pública: mudanças
  exigem versionamento e teste de regressão.

## Convenções

- Docs de design em `docs/`, em português, numerados na ordem de leitura.
- Contratos herdados dos dotfiles que este repo deve honrar:
  - Trailer `repo: ` + backtick + `host/owner/repo` + backtick em linha própria no body da issue
    (última ocorrência vence — specs podem citar o formato como exemplo).
  - Trailer `agent-config: ` com `model=<m> effort=<e>` — único registro durável de
    modelo/effort; precedência env > trailer > default é DEBUG, não fluxo normal.
  - Ledger JSONL: event-log apendado por quem gera o fato, em
    `~/.local/state/work-os/` (dados nunca ficam em repo de política).
  - Fila: prioridade Linear (urgent→low, sem-prioridade por último), depois número
    da key — computada num único lugar, painéis apenas renderizam.
- Scripts zsh em `bin/` (`work`, `attn`, `wos`, `triage`, `linear-sync`, `ship`)
  são a implementação vigente até a graduação para `src/wos/` — ao portar lógica,
  escrever primeiro o teste de regressão do comportamento real deles (incluindo o
  caso HEN-37: trailer citado como exemplo no body). Instalação: `make link`
  (symlinks em `~/.local/bin`); os bindings tmux/nvim vivem nos dotfiles.

## O que NÃO fazer

- Não criar um segundo tracker (estado próprio de item, "lido/adiado" de domínio).
- Não guardar dados de execução neste repo (vão para `~/.local/state/work-os/`).
- Não usar `orca orchestration task-list` como tracker — tasks Orca são âncoras
  efêmeras de gates.
- Não notificar em sinais ambíguos: push só em gate/escalation até o estado
  `waiting` distinguir "perguntou" de "ocioso".

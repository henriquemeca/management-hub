# 04 — Meta-loop (colher aprendizados e iterar o sistema)

Objetivo: "modelos e esforço ajustados automaticamente" e "colher aprendizados para
iterar o sistema". Isso é impossível sem dados — nenhuma política melhora sem
registro de resultado. E é perigoso sem guard-rails — um loop que se auto-modifica
sem revisão é deriva com autoridade.

## Ledger: event-log, não snapshot

Cada escritor appenda **1 linha JSONL no momento em que o fato nasce**, em
`~/.local/state/work-os/ledger.jsonl` (dados nunca vivem em repo de política):

```jsonl
{"ev":"triage",  "key":"HEN-42", "round":2, "scores":{...}, "verdict":"ready", "ts":...}
{"ev":"spawn",   "key":"HEN-42", "model":"sonnet", "effort":"medium", "policy_version":"a1b2c3d", "ts":...}
{"ev":"escalate","key":"HEN-42", "from":"sonnet/medium", "to":"opus/high", "reason":"...", "ts":...}
{"ev":"ship",    "key":"HEN-42", "lessons_ok":true, "ts":...}
```

- `policy_version` = short-hash do commit dos dotfiles/deste repo no spawn.
  "Experimento" = commitar a mudança de política, trabalhar uma semana, comparar
  entries por versão — antes/depois honesto, zero framework.
- Duração e rodadas são deriváveis por key. **Custo fica fora do MVP** até existir
  hook que o capture — melhor ausente que ficção no painel.
- Com throughput de dev solo (cap=2), números são anedota, não estatística: a retro
  prioriza o texto das lessons sobre comparações numéricas.

## Lessons: 3 perguntas fechadas, validadas no ship

O contrato do agente ao fechar a task exige comentário `## Lessons` com:

1. O que a spec não cobriu e você teve que decidir?
2. Que gate/pergunta você abriu — ou deveria ter aberto?
3. Modelo/effort foram adequados? (sim/não e por quê)

Perguntas fechadas forçam conteúdo acionável mesmo de um modelo pequeno. O `ship`
valida a presença (registra `lessons_ok` no ledger) — enforcement visível, sem
bloquear o fechamento.

## Retro: comando manual, uma proposta por vez

- `retro` é comando que o dono roda **quando sente fricção** — não cron. Automation
  mensal produz issues frias sobre contexto evaporado.
- Lê ledger + lessons **desde o último evento `retro` do próprio ledger** (o marco
  deriva do artefato — sem estado "agregado/não agregado" no hub).
- Propõe **no máximo 1 issue** de melhoria via `work` no projeto work-os — a
  proposta entra no mesmo pipeline que qualquer ideia (triagem, fila, spawn).

## Guard-rails (a parte que evita deriva)

1. **Política nunca entra no spawn automático.** Issues que tocam rubrica,
   prompt-contrato ou tabela de alocação recebem veredito forçado `escalate` na
   triagem e o `linear-sync` as ignora. O agente pode PROPOR o diff; o merge de
   política é sempre humano. (Sem isso: retro→issue→spawn→squash-merge autônomo na
   dev = um LLM reescrevendo a política que governa todos os agentes futuros, com
   feedback positivo de degradação.)
2. **Smoke test de política** como gate de PR: fixture de issue (com linha `repo:`)
   + `triage --dry-run` validando que o JSON do veredito parseia, os campos exatos
   existem, a linha `repo:` sobrevive na spec e o prompt-contrato mantém as strings
   que os parsers esperam. Política como dado sem teste = código sem teste com
   poder sobre todos os agentes futuros.
3. **Skills vs. código**: a metodologia (rubrica de interview, contrato do agente)
   é conteúdo versionado único — script e skill leem do mesmo arquivo; duplicar em
   heredoc diverge.

## Alocação automática de modelo/effort (v1)

A triagem já computa os sinais (ambiguidade numérica, rodadas, decompose). No
veredito `ready`, ela deriva e grava o trailer `agent-config` na description
(mapeamento determinístico; o LLM sugere `complexity`, o código valida):

| Sinal | Alocação |
|---|---|
| rodada 1, ambiguidade muito baixa, escopo mecânico | haiku/low |
| caso padrão | sonnet/medium |
| 2+ rodadas de interview, veio de decompose, constraints/criteria fracos | sonnet/high ou opus/high |

Humano continua soberano (picker sobrescreve o trailer antes do spawn). Escalação
manual de um toque (re-spawn com modelo maior + trailer + evento no ledger) vem
antes de qualquer escalação automática — primeiro aprender quais travadas pedem
upgrade de modelo vs. resposta humana.

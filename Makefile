BIN_DIR := $(HOME)/.local/bin
SCRIPTS := work attn wos-sh triage linear-sync ship linear-gql gates wos-hook

.PHONY: build test link unlink check

## build: compila a CLI Go em dist/wos (não conflita com bin/wos zsh até a graduação)
build:
	go build -o dist/wos .

test:
	go test ./...

## link: compila e symlinka tudo em ~/.local/bin — `wos` aponta para o binário
## Go (TUI + passthrough para wos-sh); os demais são os scripts zsh de bin/
link: build
	@mkdir -p $(BIN_DIR)
	@ln -sfn $(CURDIR)/dist/wos $(BIN_DIR)/wos
	@echo "  $(BIN_DIR)/wos -> dist/wos (TUI + passthrough)"
	@for s in $(SCRIPTS); do \
		ln -sfn $(CURDIR)/bin/$$s $(BIN_DIR)/$$s; \
		echo "  $(BIN_DIR)/$$s -> bin/$$s"; \
	done

unlink:
	@for s in $(SCRIPTS); do \
		[ -L $(BIN_DIR)/$$s ] && rm $(BIN_DIR)/$$s && echo "  removido $(BIN_DIR)/$$s" || true; \
	done

## check: cada script responde a --help (smoke de sanidade pós-link)
check:
	@for s in $(SCRIPTS); do \
		$(BIN_DIR)/$$s --help >/dev/null 2>&1 && echo "  ok $$s" || echo "  FALHOU $$s"; \
	done

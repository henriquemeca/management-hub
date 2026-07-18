BIN_DIR := $(HOME)/.local/bin
SCRIPTS := work attn wos triage linear-sync ship

.PHONY: link unlink check

## link: symlinka bin/* em ~/.local/bin (caminhos que tmux/nvim/automations já usam)
link:
	@mkdir -p $(BIN_DIR)
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

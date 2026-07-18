package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/henriquemeca/management-hub/commands"
)

// goCommands são os subcomandos servidos pelo binário Go; o resto (spawn, ps,
// peek, say, go, doctor, capture, triage, sync, ship, attn…) é passthrough
// para o bin/wos-sh — a implementação zsh vigente até a graduação.
var goCommands = map[string]bool{
	"ui":               true,
	"feed":             true,
	"version":          true,
	"help":             true,
	"-h":               true,
	"--help":           true,
	"completion":       true,
	"__complete":       true, // protocolo interno de completion do cobra
	"__completeNoDesc": true,
}

func main() {
	if len(os.Args) > 1 && !goCommands[os.Args[1]] {
		passthrough(os.Args[1:])
	}
	commands.Execute()
}

// passthrough substitui o processo pelo wos-sh (execve), repassando os args.
func passthrough(args []string) {
	sh := findWosSH()
	if sh == "" {
		fmt.Fprintln(os.Stderr, "wos: bin/wos-sh não encontrado (nem ao lado do binário nem em ~/.local/bin)")
		os.Exit(1)
	}
	argv := append([]string{sh}, args...)
	if err := syscall.Exec(sh, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "wos: exec %s: %v\n", sh, err)
		os.Exit(1)
	}
}

// findWosSH resolve o wos-sh ao lado do binário (dist/wos → ../bin/wos-sh),
// com fallback em ~/.local/bin/wos-sh.
func findWosSH() string {
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil {
			self = resolved
		}
		p := filepath.Join(filepath.Dir(self), "..", "bin", "wos-sh")
		if isExecutable(p) {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".local", "bin", "wos-sh")
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

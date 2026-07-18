package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestFeedEmitsValidVersionedJSON(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()

	if err := feedCmd.RunE(feedCmd, nil); err != nil {
		t.Fatalf("feed retornou erro: %v", err)
	}
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}

	var feed Feed
	if err := json.Unmarshal(buf.Bytes(), &feed); err != nil {
		t.Fatalf("saída não é JSON válido: %v\n%s", err, buf.String())
	}
	if feed.Version != 1 {
		t.Errorf("version = %d, esperado 1", feed.Version)
	}
	if feed.Items == nil {
		t.Error("items deve ser lista (possivelmente vazia), nunca null")
	}
}

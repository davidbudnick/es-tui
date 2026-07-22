package main

import (
	"errors"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/ui"
)

type fakeProgram struct {
	err error
}

func (f *fakeProgram) Send(msg tea.Msg) {}
func (f *fakeProgram) Run() (tea.Model, error) {
	return nil, f.err
}

func TestProdLogFatalAndRunApp(t *testing.T) {
	old := logFatalf
	called := false
	logFatalf = func(v ...any) { called = true }
	t.Cleanup(func() { logFatalf = old })
	prodLogFatal("x")
	if !called {
		t.Fatal("prodLogFatal")
	}

	oldNP := newProgram
	newProgram = func(m ui.Model) teaProgram {
		return &fakeProgram{}
	}
	t.Cleanup(func() { newProgram = oldNP })

	m := ui.NewModel()
	send := func(tea.Msg) {}
	m.SendFunc = &send
	if err := prodRunApp(m); err != nil {
		t.Fatal(err)
	}

	newProgram = func(m ui.Model) teaProgram {
		return &fakeProgram{err: errors.New("run")}
	}
	if err := prodRunApp(m); err == nil {
		t.Fatal("expected error")
	}

	m.SendFunc = nil
	newProgram = func(m ui.Model) teaProgram { return &fakeProgram{} }
	if err := prodRunApp(m); err != nil {
		t.Fatal(err)
	}
}

func TestParseCLIFlagsHost(t *testing.T) {
	oldArgs := os.Args
	oldExit := osExit
	osExit = func(int) {}
	os.Args = []string{"es-tui", "--host", "localhost", "--port", "9200"}
	t.Cleanup(func() {
		os.Args = oldArgs
		osExit = oldExit
	})
	conn := parseCLIFlags()
	if conn == nil || conn.Host != "localhost" {
		t.Fatal(conn)
	}
}

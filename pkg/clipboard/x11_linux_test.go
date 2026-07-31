//go:build linux

package clipboard

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func TestX11ClipboardServesHTMLAndPlainText(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("X11 display is not available")
	}

	ownerDone := make(chan error, 1)
	go func() {
		ownerDone <- serveX11Clipboard(
			`<html><body><a href="https://example.com/pr/1">PR #1</a></body></html>`,
			"PR #1 (https://example.com/pr/1)",
		)
	}()

	connection, err := xgb.NewConn()
	if err != nil {
		t.Fatalf("connect to X11: %v", err)
	}
	defer connection.Close()

	atoms, err := internX11Atoms(connection)
	if err != nil {
		t.Fatalf("intern atoms: %v", err)
	}

	requestor, err := xproto.NewWindowId(connection)
	if err != nil {
		t.Fatalf("allocate requestor window: %v", err)
	}
	screen := xproto.Setup(connection).DefaultScreen(connection)
	if err := xproto.CreateWindowChecked(
		connection,
		screen.RootDepth,
		requestor,
		screen.Root,
		0,
		0,
		1,
		1,
		0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,
		0,
		nil,
	).Check(); err != nil {
		t.Fatalf("create requestor window: %v", err)
	}
	defer xproto.DestroyWindow(connection, requestor)

	property, err := internTestAtom(connection)
	if err != nil {
		t.Fatalf("intern test property: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		owner, ownerErr := xproto.GetSelectionOwner(connection, atoms.clipboard).Reply()
		if ownerErr != nil {
			t.Fatalf("get selection owner: %v", ownerErr)
		}
		if owner != nil && owner.Owner != xproto.WindowNone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clipboard owner did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	html := requestX11Target(t, connection, requestor, atoms.clipboard, atoms.html, property)
	if !strings.Contains(html, `>PR #1</a>`) {
		t.Fatalf("HTML clipboard payload = %q, want PR anchor", html)
	}

	for name, target := range map[string]xproto.Atom{
		"UTF8_STRING": atoms.utf8,
		"text/plain":  atoms.plain,
	} {
		plain := requestX11Target(t, connection, requestor, atoms.clipboard, target, property)
		if plain != "PR #1 (https://example.com/pr/1)" {
			t.Fatalf("%s clipboard payload = %q, want plain report", name, plain)
		}
	}

	if err := xproto.SetSelectionOwnerChecked(connection, xproto.WindowNone, atoms.clipboard, xproto.TimeCurrentTime).Check(); err != nil {
		t.Fatalf("release clipboard: %v", err)
	}

	select {
	case err := <-ownerDone:
		if err != nil {
			t.Fatalf("clipboard owner: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("clipboard owner did not stop after losing selection")
	}
}

func internTestAtom(connection *xgb.Conn) (xproto.Atom, error) {
	name := "ADOCTL_TEST_CLIPBOARD_PROPERTY"
	reply, err := xproto.InternAtom(connection, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, fmt.Errorf("intern atom returned no result")
	}
	return reply.Atom, nil
}

func requestX11Target(t *testing.T, connection *xgb.Conn, requestor xproto.Window, selection, target, property xproto.Atom) string {
	t.Helper()
	if err := xproto.ConvertSelectionChecked(connection, requestor, selection, target, property, xproto.TimeCurrentTime).Check(); err != nil {
		t.Fatalf("request target %d: %v", target, err)
	}

	for {
		event, eventErr := connection.WaitForEvent()
		if eventErr != nil {
			t.Fatalf("wait for target %d: %v", target, eventErr)
		}
		selectionEvent, ok := event.(xproto.SelectionNotifyEvent)
		if !ok || selectionEvent.Property == xproto.AtomNone {
			continue
		}
		reply, err := xproto.GetProperty(connection, true, requestor, property, xproto.AtomAny, 0, 1024*1024).Reply()
		if err != nil {
			t.Fatalf("read target %d: %v", target, err)
		}
		return string(reply.Value)
	}
}

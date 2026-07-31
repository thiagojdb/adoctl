//go:build linux

package clipboard

import (
	"bytes"
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
	if err := xproto.ChangeWindowAttributesChecked(
		connection,
		requestor,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskPropertyChange},
	).Check(); err != nil {
		t.Fatalf("select requestor property events: %v", err)
	}

	property, err := internTestAtom(connection)
	if err != nil {
		t.Fatalf("intern test property: %v", err)
	}

	initialOwnerReply, err := xproto.GetSelectionOwner(connection, atoms.clipboard).Reply()
	if err != nil {
		t.Fatalf("get initial selection owner: %v", err)
	}
	initialOwner := xproto.Window(xproto.WindowNone)
	if initialOwnerReply != nil {
		initialOwner = initialOwnerReply.Owner
	}

	largeHTML := `<html><body><a href="https://example.com/pr/1">PR #1</a>` + strings.Repeat("x", 300000) + `</body></html>`
	ownerDone := make(chan error, 1)
	go func() {
		ownerDone <- serveX11Clipboard(
			largeHTML,
			"PR #1 (https://example.com/pr/1)",
		)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		owner, ownerErr := xproto.GetSelectionOwner(connection, atoms.clipboard).Reply()
		if ownerErr != nil {
			t.Fatalf("get selection owner: %v", ownerErr)
		}
		if owner != nil && owner.Owner != xproto.WindowNone && owner.Owner != initialOwner {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clipboard owner did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	html := requestX11Target(t, connection, requestor, atoms.clipboard, atoms.html, property, atoms.incr)
	if !strings.Contains(html, `>PR #1</a>`) {
		t.Fatalf("HTML clipboard payload = %q, want PR anchor", html)
	}

	for name, target := range map[string]xproto.Atom{
		"UTF8_STRING": atoms.utf8,
		"text/plain":  atoms.plain,
	} {
		plain := requestX11Target(t, connection, requestor, atoms.clipboard, target, property, atoms.incr)
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

func requestX11Target(t *testing.T, connection *xgb.Conn, requestor xproto.Window, selection, target, property, incr xproto.Atom) string {
	t.Helper()
	if err := xproto.ConvertSelectionChecked(connection, requestor, selection, target, property, xproto.TimeCurrentTime).Check(); err != nil {
		t.Fatalf("request target %d: %v", target, err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		event, eventErr := connection.PollForEvent()
		if eventErr != nil {
			t.Fatalf("wait for target %d: %v", target, eventErr)
		}
		if event == nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		selectionEvent, ok := event.(xproto.SelectionNotifyEvent)
		if !ok {
			continue
		}
		if selectionEvent.Property == xproto.AtomNone {
			t.Fatalf("target %d was refused by clipboard owner", target)
		}
		reply, err := xproto.GetProperty(connection, true, requestor, property, xproto.AtomAny, 0, 1024*1024).Reply()
		if err != nil {
			t.Fatalf("read target %d: %v", target, err)
		}
		if reply == nil {
			t.Fatalf("read target %d returned no property", target)
		}
		if reply.Type != incr {
			return string(reply.Value)
		}

		var data bytes.Buffer
		for time.Now().Before(deadline) {
			event, eventErr := connection.PollForEvent()
			if eventErr != nil {
				t.Fatalf("wait for target %d chunk: %v", target, eventErr)
			}
			if event == nil {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			propertyEvent, ok := event.(xproto.PropertyNotifyEvent)
			if !ok || propertyEvent.Window != requestor || propertyEvent.Atom != property || propertyEvent.State != xproto.PropertyNewValue {
				continue
			}
			chunk, err := xproto.GetProperty(connection, true, requestor, property, xproto.AtomAny, 0, 1024*1024).Reply()
			if err != nil {
				t.Fatalf("read target %d chunk: %v", target, err)
			}
			if chunk == nil {
				t.Fatalf("read target %d chunk returned no property", target)
			}
			if len(chunk.Value) == 0 {
				return data.String()
			}
			_, _ = data.Write(chunk.Value)
		}
		t.Fatalf("timed out reading target %d chunks", target)
	}
	t.Fatalf("timed out waiting for target %d", target)
	return ""
}

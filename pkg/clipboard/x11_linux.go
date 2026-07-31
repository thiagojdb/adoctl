//go:build linux

package clipboard

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// serveX11Clipboard owns the X11 CLIPBOARD selection and serves both rich and
// plain-text targets. It intentionally blocks until another application takes
// ownership; the caller runs it in the detached clipboard-server process.
func serveX11Clipboard(html, plain string) error {
	connection, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("x11: connect to display: %w", err)
	}
	defer connection.Close()

	screen := xproto.Setup(connection).DefaultScreen(connection)
	window, err := xproto.NewWindowId(connection)
	if err != nil {
		return fmt.Errorf("x11: allocate clipboard window: %w", err)
	}

	if err := xproto.CreateWindowChecked(
		connection,
		screen.RootDepth,
		window,
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
		return fmt.Errorf("x11: create clipboard window: %w", err)
	}
	defer xproto.DestroyWindow(connection, window)

	atoms, err := internX11Atoms(connection)
	if err != nil {
		return err
	}

	if err := xproto.SetSelectionOwnerChecked(connection, window, atoms.clipboard, xproto.TimeCurrentTime).Check(); err != nil {
		return fmt.Errorf("x11: claim clipboard: %w", err)
	}

	owner, err := xproto.GetSelectionOwner(connection, atoms.clipboard).Reply()
	if err != nil {
		return fmt.Errorf("x11: verify clipboard ownership: %w", err)
	}
	if owner == nil || owner.Owner != window {
		return fmt.Errorf("x11: clipboard ownership was not acquired")
	}

	for {
		event, eventErr := connection.WaitForEvent()
		if eventErr != nil {
			return fmt.Errorf("x11: clipboard event: %w", eventErr)
		}
		if event == nil {
			return fmt.Errorf("x11: clipboard connection closed")
		}

		switch event := event.(type) {
		case xproto.SelectionClearEvent:
			return nil
		case xproto.SelectionRequestEvent:
			serveX11SelectionRequest(connection, atoms, event, html, plain)
		}
	}
}

type x11Atoms struct {
	clipboard  xproto.Atom
	targets    xproto.Atom
	html       xproto.Atom
	plain      xproto.Atom
	plainUTF8  xproto.Atom
	utf8       xproto.Atom
	stringAtom xproto.Atom
	text       xproto.Atom
}

func internX11Atoms(connection *xgb.Conn) (x11Atoms, error) {
	atom := func(name string) (xproto.Atom, error) {
		reply, err := xproto.InternAtom(connection, false, uint16(len(name)), name).Reply()
		if err != nil {
			return 0, fmt.Errorf("x11: intern atom %q: %w", name, err)
		}
		if reply == nil {
			return 0, fmt.Errorf("x11: intern atom %q returned no result", name)
		}
		return reply.Atom, nil
	}

	clipboard, err := atom("CLIPBOARD")
	if err != nil {
		return x11Atoms{}, err
	}
	targets, err := atom("TARGETS")
	if err != nil {
		return x11Atoms{}, err
	}
	html, err := atom("text/html")
	if err != nil {
		return x11Atoms{}, err
	}
	plain, err := atom("text/plain")
	if err != nil {
		return x11Atoms{}, err
	}
	plainUTF8, err := atom("text/plain;charset=utf-8")
	if err != nil {
		return x11Atoms{}, err
	}
	utf8, err := atom("UTF8_STRING")
	if err != nil {
		return x11Atoms{}, err
	}
	text, err := atom("TEXT")
	if err != nil {
		return x11Atoms{}, err
	}

	return x11Atoms{
		clipboard:  clipboard,
		targets:    targets,
		html:       html,
		plain:      plain,
		plainUTF8:  plainUTF8,
		utf8:       utf8,
		stringAtom: xproto.AtomString,
		text:       text,
	}, nil
}

func serveX11SelectionRequest(connection *xgb.Conn, atoms x11Atoms, request xproto.SelectionRequestEvent, html, plain string) {
	property := request.Property
	if property == xproto.AtomNone {
		property = request.Target
	}

	notify := xproto.SelectionNotifyEvent{
		Time:      request.Time,
		Requestor: request.Requestor,
		Selection: request.Selection,
		Target:    request.Target,
		Property:  xproto.AtomNone,
	}

	switch request.Target {
	case atoms.targets:
		targets := []xproto.Atom{
			atoms.targets,
			atoms.html,
			atoms.plainUTF8,
			atoms.plain,
			atoms.utf8,
			atoms.stringAtom,
			atoms.text,
		}
		xproto.ChangeProperty(
			connection,
			xproto.PropModeReplace,
			request.Requestor,
			property,
			xproto.AtomAtom,
			32,
			uint32(len(targets)),
			x11AtomBytes(targets),
		)
		notify.Property = property
	case atoms.html:
		xproto.ChangeProperty(
			connection,
			xproto.PropModeReplace,
			request.Requestor,
			property,
			atoms.html,
			8,
			uint32(len(html)),
			[]byte(html),
		)
		notify.Property = property
	case atoms.plainUTF8, atoms.plain, atoms.utf8, atoms.stringAtom, atoms.text:
		typeAtom := atoms.utf8
		if request.Target == atoms.stringAtom {
			typeAtom = atoms.stringAtom
		}
		xproto.ChangeProperty(
			connection,
			xproto.PropModeReplace,
			request.Requestor,
			property,
			typeAtom,
			8,
			uint32(len(plain)),
			[]byte(plain),
		)
		notify.Property = property
	}

	xproto.SendEvent(connection, false, request.Requestor, 0, string(notify.Bytes()))
}

func x11AtomBytes(atoms []xproto.Atom) []byte {
	data := make([]byte, len(atoms)*4)
	for i, atom := range atoms {
		xgb.Put32(data[i*4:], uint32(atom))
	}
	return data
}

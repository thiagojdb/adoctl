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

	transfers := make(map[x11TransferKey]*x11Transfer)
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
			if transfer := serveX11SelectionRequest(connection, atoms, event, html, plain); transfer != nil {
				transfers[transfer.key()] = transfer
			}
		case xproto.PropertyNotifyEvent:
			if event.State != xproto.PropertyDelete {
				continue
			}
			key := x11TransferKey{requestor: event.Window, property: event.Atom}
			transfer, ok := transfers[key]
			if !ok {
				continue
			}
			finished := transfer.offset >= len(transfer.data)
			if err := transfer.sendNext(connection); err != nil {
				delete(transfers, key)
			} else if finished {
				delete(transfers, key)
			}
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
	incr       xproto.Atom
}

// internX11Atoms resolves the selection and target atoms used by the server.
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
	incr, err := atom("INCR")
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
		incr:       incr,
	}, nil
}

type x11TransferKey struct {
	requestor xproto.Window
	property  xproto.Atom
}

type x11Transfer struct {
	transferKey x11TransferKey
	typeAtom    xproto.Atom
	data        []byte
	offset      int
}

// sendNext publishes the next chunk of an ICCCM INCR transfer.
func (transfer *x11Transfer) sendNext(connection *xgb.Conn) error {
	const chunkSize = 64 * 1024
	if transfer.offset >= len(transfer.data) {
		return xproto.ChangePropertyChecked(
			connection,
			xproto.PropModeReplace,
			transfer.transferKey.requestor,
			transfer.transferKey.property,
			transfer.typeAtom,
			8,
			0,
			nil,
		).Check()
	}

	end := transfer.offset + chunkSize
	if end > len(transfer.data) {
		end = len(transfer.data)
	}
	chunk := transfer.data[transfer.offset:end]
	if err := xproto.ChangePropertyChecked(
		connection,
		xproto.PropModeReplace,
		transfer.transferKey.requestor,
		transfer.transferKey.property,
		transfer.typeAtom,
		8,
		uint32(len(chunk)),
		chunk,
	).Check(); err != nil {
		return err
	}
	transfer.offset = end
	return nil
}

// key returns the requestor/property pair identifying this transfer.
func (transfer *x11Transfer) key() x11TransferKey {
	return transfer.transferKey
}

// serveX11SelectionRequest answers a request for one of the advertised targets.
func serveX11SelectionRequest(connection *xgb.Conn, atoms x11Atoms, request xproto.SelectionRequestEvent, html, plain string) *x11Transfer {
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

	var transfer *x11Transfer
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
		if err := xproto.ChangePropertyChecked(
			connection,
			xproto.PropModeReplace,
			request.Requestor,
			property,
			xproto.AtomAtom,
			32,
			uint32(len(targets)),
			x11AtomBytes(targets),
		).Check(); err == nil {
			notify.Property = property
		}
	case atoms.html:
		notify.Property, transfer = setX11ClipboardProperty(connection, atoms, request.Requestor, property, atoms.html, []byte(html))
	case atoms.plainUTF8, atoms.plain, atoms.utf8, atoms.stringAtom, atoms.text:
		typeAtom := atoms.utf8
		if request.Target == atoms.stringAtom {
			typeAtom = atoms.stringAtom
		}
		notify.Property, transfer = setX11ClipboardProperty(connection, atoms, request.Requestor, property, typeAtom, []byte(plain))
	}

	_ = xproto.SendEventChecked(connection, false, request.Requestor, 0, string(notify.Bytes())).Check()
	return transfer
}

// setX11ClipboardProperty writes a payload directly or starts an INCR transfer.
func setX11ClipboardProperty(connection *xgb.Conn, atoms x11Atoms, requestor xproto.Window, property, typeAtom xproto.Atom, data []byte) (xproto.Atom, *x11Transfer) {
	if uint64(len(data)) > uint64(^uint32(0)) {
		return xproto.AtomNone, nil
	}

	if len(data) <= x11MaxPropertyBytes(connection) {
		if err := xproto.ChangePropertyChecked(
			connection,
			xproto.PropModeReplace,
			requestor,
			property,
			typeAtom,
			8,
			uint32(len(data)),
			data,
		).Check(); err == nil {
			return property, nil
		}
		return xproto.AtomNone, nil
	}

	total := make([]byte, 4)
	xgb.Put32(total, uint32(len(data)))
	if err := xproto.ChangePropertyChecked(
		connection,
		xproto.PropModeReplace,
		requestor,
		property,
		atoms.incr,
		32,
		1,
		total,
	).Check(); err != nil {
		return xproto.AtomNone, nil
	}
	if err := xproto.ChangeWindowAttributesChecked(
		connection,
		requestor,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskPropertyChange},
	).Check(); err != nil {
		return xproto.AtomNone, nil
	}

	return property, &x11Transfer{
		transferKey: x11TransferKey{requestor: requestor, property: property},
		typeAtom:    typeAtom,
		data:        data,
	}
}

// x11MaxPropertyBytes returns a safe single-request payload limit.
func x11MaxPropertyBytes(connection *xgb.Conn) int {
	// ChangeProperty's fixed request header is 24 bytes. Leave additional room
	// for padding and protocol variation; large payloads use ICCCM INCR.
	max := int(xproto.Setup(connection).MaximumRequestLength)*4 - 64
	if max < 1 {
		return 1
	}
	return max
}

// x11AtomBytes encodes atoms in the 32-bit wire format used by TARGETS.
func x11AtomBytes(atoms []xproto.Atom) []byte {
	data := make([]byte, len(atoms)*4)
	for i, atom := range atoms {
		xgb.Put32(data[i*4:], uint32(atom))
	}
	return data
}

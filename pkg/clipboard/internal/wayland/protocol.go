package wayland

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var le = binary.LittleEndian

// Fixed Wayland object IDs we assign (client-range: 2–0xfeffffff).
const (
	idDisplay   uint32 = 1
	idRegistry  uint32 = 2
	idCallback1 uint32 = 3 // first sync
	idSeat      uint32 = 4
	idDCManager uint32 = 5 // zwlr_data_control_manager_v1
	idDCSource  uint32 = 6 // zwlr_data_control_source_v1
	idDCDevice  uint32 = 7 // zwlr_data_control_device_v1
	idCallback2 uint32 = 8 // second sync
)

// waylandConn is a buffered Wayland connection.
type waylandConn struct {
	fd         int
	inBuf      []byte
	pendingFds []int
}

func newConn(sockPath string) (*waylandConn, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	if err := syscall.Connect(fd, &syscall.SockaddrUnix{Name: sockPath}); err != nil {
		syscall.Close(fd) //nolint:errcheck
		return nil, err
	}
	return &waylandConn{fd: fd}, nil
}

func (c *waylandConn) close() {
	syscall.Close(c.fd) //nolint:errcheck
}

// sendMsg sends a Wayland request message.
func (c *waylandConn) sendMsg(objectID uint32, opcode uint16, args []byte) error {
	size := uint16(8 + len(args))
	buf := make([]byte, size)
	le.PutUint32(buf[0:], objectID)
	le.PutUint32(buf[4:], uint32(opcode)|uint32(size)<<16)
	copy(buf[8:], args)
	_, err := syscall.Write(c.fd, buf)
	return err
}

// readMsg reads the next complete Wayland event, returning any fd from SCM_RIGHTS.
// fd is -1 if no file descriptor was delivered with this message.
func (c *waylandConn) readMsg() (objectID uint32, opcode uint16, payload []byte, fd int, err error) {
	fd = -1
	for {
		if len(c.inBuf) >= 8 {
			sizeOpcode := le.Uint32(c.inBuf[4:8])
			size := int(sizeOpcode >> 16)
			if size >= 8 && len(c.inBuf) >= size {
				objectID = le.Uint32(c.inBuf[0:4])
				opcode = uint16(sizeOpcode & 0xffff)
				payload = make([]byte, size-8)
				copy(payload, c.inBuf[8:size])
				c.inBuf = c.inBuf[size:]
				if len(c.pendingFds) > 0 {
					fd = c.pendingFds[0]
					c.pendingFds = c.pendingFds[1:]
				}
				return
			}
		}

		// Read more data from socket.
		buf := make([]byte, 4096)
		oob := make([]byte, syscall.CmsgSpace(4*8)) // room for up to 8 fds
		n, oobn, _, _, recvErr := syscall.Recvmsg(c.fd, buf, oob, 0)
		if recvErr != nil {
			err = recvErr
			return
		}
		if n == 0 {
			err = fmt.Errorf("wayland: connection closed")
			return
		}
		c.inBuf = append(c.inBuf, buf[:n]...)

		if oobn > 0 {
			scms, parseErr := syscall.ParseSocketControlMessage(oob[:oobn])
			if parseErr == nil {
				for _, scm := range scms {
					rights, parseErr := syscall.ParseUnixRights(&scm)
					if parseErr == nil {
						c.pendingFds = append(c.pendingFds, rights...)
					}
				}
			}
		}
	}
}

func encodeUint32(v uint32) []byte {
	b := make([]byte, 4)
	le.PutUint32(b, v)
	return b
}

// encodeString encodes a Wayland string: uint32 length (incl. null), bytes, padding to 4-byte alignment.
func encodeString(s string) []byte {
	sBytes := append([]byte(s), 0) // null terminator
	length := len(sBytes)
	padded := (length + 3) &^ 3
	buf := make([]byte, 4+padded)
	le.PutUint32(buf[0:], uint32(length))
	copy(buf[4:], sBytes)
	return buf
}

func concat(slices ...[]byte) []byte {
	var total int
	for _, s := range slices {
		total += len(s)
	}
	result := make([]byte, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// decodeString reads a Wayland string from payload bytes.
func decodeString(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", data, fmt.Errorf("wayland: short string length field")
	}
	length := int(le.Uint32(data[:4]))
	data = data[4:]
	if length == 0 {
		return "", data, nil
	}
	padded := (length + 3) &^ 3
	if len(data) < padded {
		return "", data, fmt.Errorf("wayland: short string data")
	}
	s := string(data[:length-1]) // exclude null terminator
	return s, data[padded:], nil
}

// Serve claims the Wayland clipboard (zwlr_data_control_v1) and blocks until
// ownership is cancelled by another clipboard write. It serves each MIME type
// on demand by writing the corresponding bytes to the fd provided by the compositor.
func Serve(formats map[string][]byte) error {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		display = "wayland-0"
	}
	if runtime == "" {
		return fmt.Errorf("wayland: XDG_RUNTIME_DIR not set")
	}

	sockPath := filepath.Join(runtime, display)
	c, err := newConn(sockPath)
	if err != nil {
		return fmt.Errorf("wayland: connect %s: %w", sockPath, err)
	}
	defer c.close()

	// 1. Request registry.
	if err := c.sendMsg(idDisplay, 1 /*get_registry*/, encodeUint32(idRegistry)); err != nil {
		return err
	}

	// 2. Sync to flush globals.
	if err := c.sendMsg(idDisplay, 0 /*sync*/, encodeUint32(idCallback1)); err != nil {
		return err
	}

	// 3. Collect globals until callback done.
	var seatName, dcManagerName uint32
	var seatFound, dcManagerFound bool

	for {
		objectID, opcode, payload, fd, err := c.readMsg()
		if err != nil {
			return err
		}
		if fd >= 0 {
			syscall.Close(fd) //nolint:errcheck
		}

		switch {
		case objectID == idRegistry && opcode == 0 /*global*/:
			if len(payload) < 4 {
				continue
			}
			name := le.Uint32(payload[:4])
			iface, _, decErr := decodeString(payload[4:])
			if decErr != nil {
				continue
			}
			switch iface {
			case "wl_seat":
				seatName = name
				seatFound = true
			case "zwlr_data_control_manager_v1":
				dcManagerName = name
				dcManagerFound = true
			}

		case objectID == idCallback1 && opcode == 0 /*done*/:
			goto afterFirstSync
		}
	}

afterFirstSync:
	if !seatFound {
		return fmt.Errorf("wayland: wl_seat not found")
	}
	if !dcManagerFound {
		return fmt.Errorf("wayland: zwlr_data_control_manager_v1 not found (compositor may not support wlr-data-control)")
	}

	// 4. Bind wl_seat.
	// wl_registry.bind new_id encodes inline: [name][interface string][version][new_id]
	if err := c.sendMsg(idRegistry, 0 /*bind*/, concat(
		encodeUint32(seatName),
		encodeString("wl_seat"),
		encodeUint32(1),
		encodeUint32(idSeat),
	)); err != nil {
		return err
	}

	// 5. Bind zwlr_data_control_manager_v1.
	if err := c.sendMsg(idRegistry, 0 /*bind*/, concat(
		encodeUint32(dcManagerName),
		encodeString("zwlr_data_control_manager_v1"),
		encodeUint32(2),
		encodeUint32(idDCManager),
	)); err != nil {
		return err
	}

	// 6. Create data source.
	if err := c.sendMsg(idDCManager, 0 /*create_data_source*/, encodeUint32(idDCSource)); err != nil {
		return err
	}

	// 7. Offer each MIME type.
	for mimeType := range formats {
		if err := c.sendMsg(idDCSource, 0 /*offer*/, encodeString(mimeType)); err != nil {
			return err
		}
	}

	// 8. Get data device.
	if err := c.sendMsg(idDCManager, 1 /*get_data_device*/, concat(
		encodeUint32(idDCDevice),
		encodeUint32(idSeat),
	)); err != nil {
		return err
	}

	// 9. Set selection.
	if err := c.sendMsg(idDCDevice, 0 /*set_selection*/, encodeUint32(idDCSource)); err != nil {
		return err
	}

	// 10. Second sync to confirm ownership.
	if err := c.sendMsg(idDisplay, 0 /*sync*/, encodeUint32(idCallback2)); err != nil {
		return err
	}

	for {
		objectID, opcode, _, fd, err := c.readMsg()
		if err != nil {
			return err
		}
		if fd >= 0 {
			syscall.Close(fd) //nolint:errcheck
		}
		if objectID == idCallback2 && opcode == 0 /*done*/ {
			break
		}
	}

	// 11. Event loop: serve paste requests until ownership is cancelled.
	for {
		objectID, opcode, payload, fd, err := c.readMsg()
		if err != nil {
			// Connection closed means compositor exited; treat as done.
			return nil
		}

		if objectID != idDCSource {
			if fd >= 0 {
				syscall.Close(fd) //nolint:errcheck
			}
			continue
		}

		switch opcode {
		case 0: // zwlr_data_control_source_v1.send
			mimeType, _, _ := decodeString(payload)
			if fd >= 0 {
				if data, ok := formats[mimeType]; ok {
					syscall.Write(fd, data) //nolint:errcheck
				}
				syscall.Close(fd) //nolint:errcheck
			}
		case 1: // zwlr_data_control_source_v1.cancelled
			return nil
		}
	}
}

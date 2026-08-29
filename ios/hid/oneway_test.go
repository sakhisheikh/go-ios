package hid

import (
	"bytes"
	"io"
	"testing"

	"github.com/danielpaulus/go-ios/ios/xpc"
)

type nopXPCCloser struct{ io.ReadWriter }

func (nopXPCCloser) Close() error { return nil }

func testXPCConnection(t *testing.T) (*xpc.Connection, *bytes.Buffer) {
	t.Helper()
	buf := bytes.NewBuffer(nil)
	conn, err := xpc.New(buf, bytes.NewBuffer(nil), nopXPCCloser{ReadWriter: buf})
	if err != nil {
		t.Fatal(err)
	}
	return conn, buf
}

func assertOneWayMessage(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	msg, err := xpc.DecodeMessage(buf)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Flags&xpc.HeartbeatRequestFlag != 0 {
		t.Fatalf("one-way HID message requested an unread reply: flags=%#x", msg.Flags)
	}
}

func TestSendReportIsOneWay(t *testing.T) {
	conn, buf := testXPCConnection(t)
	u := &UniversalConnection{conn: conn}
	if err := u.SendReport(SurfaceMainTouchscreen, []byte{1}); err != nil {
		t.Fatal(err)
	}
	assertOneWayMessage(t, buf)
}

func TestSendButtonIsOneWay(t *testing.T) {
	conn, buf := testXPCConnection(t)
	i := &IndigoConnection{conn: conn}
	if err := i.SendButton(12, 64, ButtonDown); err != nil {
		t.Fatal(err)
	}
	assertOneWayMessage(t, buf)
}

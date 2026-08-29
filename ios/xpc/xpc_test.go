package xpc

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

type nopReadWriteCloser struct{ io.ReadWriter }

func (nopReadWriteCloser) Close() error { return nil }

func TestConnectionSendIncrementsMessageID(t *testing.T) {
	var sent bytes.Buffer
	conn, err := New(&sent, bytes.NewBuffer(nil), nopReadWriteCloser{ReadWriter: &sent})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Send(map[string]interface{}{"n": uint64(1)}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Send(map[string]interface{}{"n": uint64(2)}); err != nil {
		t.Fatal(err)
	}

	ids := make([]uint64, 0, 2)
	for sent.Len() > 0 {
		var magic uint32
		var header wrapperHeader
		if err := binary.Read(&sent, binary.LittleEndian, &magic); err != nil {
			t.Fatal(err)
		}
		if err := binary.Read(&sent, binary.LittleEndian, &header); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, header.MsgId)
		if _, err := io.CopyN(io.Discard, &sent, int64(header.BodyLen)); err != nil {
			t.Fatal(err)
		}
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("message IDs = %v; want [1 2]", ids)
	}
}

package hid

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// report is one call recorded by fakeHID, so a test can assert what a gesture
// put on the wire rather than what it meant to.
type report struct {
	kind  string
	state TouchState
	x, y  uint16
}

type fakeHID struct {
	reports []report
	// failAt makes the nth SendTouch call fail, to exercise the paths that
	// have to clean up after a gesture breaks part way through.
	failAt int
	calls  int
}

func (f *fakeHID) SendTouch(state TouchState, p Point) error {
	f.calls++
	if f.failAt > 0 && f.calls == f.failAt {
		return errors.New("send failed")
	}
	f.reports = append(f.reports, report{kind: "touch", state: state, x: p.X, y: p.Y})
	return nil
}

func (f *fakeHID) Close() error { return nil }

// stubStream stands in for a running media stream. ensureStream returns as soon
// as it sees one, so it never has to do anything.
type stubStream struct{ stopped bool }

func (s *stubStream) StopMediaStream(context.Context, uuid.UUID) error { s.stopped = true; return nil }
func (s *stubStream) Close() error                                     { return nil }

// openSession builds a Session with the gate already held open, which is what a
// caller has after EnsureStream.
func openSession() (*Session, *fakeHID) {
	f := &fakeHID{}
	return &Session{hid: f, displayService: &stubStream{}}, f
}

func touches(reports []report) []report {
	var out []report
	for _, r := range reports {
		if r.kind == "touch" {
			out = append(out, r)
		}
	}
	return out
}

func TestTapContactsThenReleasesAtTheSamePoint(t *testing.T) {
	session, fake := openSession()
	require.NoError(t, session.Tap(context.Background(), Point{X: 100, Y: 200}))

	got := touches(fake.reports)
	require.Len(t, got, 2)
	assert.Equal(t, TouchContact, got[0].state)
	assert.Equal(t, TouchRelease, got[1].state)
	assert.Equal(t, [2]uint16{100, 200}, [2]uint16{got[0].x, got[0].y})
	assert.Equal(t, [2]uint16{100, 200}, [2]uint16{got[1].x, got[1].y},
		"the release has to land where the contact did, or it reads as a flick")
}

func TestStrokeReleasesWhereTheContactActuallyIs(t *testing.T) {
	// The regression this guards: the release used to go to the last point of the
	// path even when a sample failed early, which the device reads as a flick to
	// somewhere the finger never travelled.
	session, fake := openSession()
	fake.failAt = 3

	err := session.Stroke(context.Background(), []Point{
		{X: 10, Y: 10}, {X: 20, Y: 20}, {X: 30, Y: 30}, {X: 900, Y: 900},
	}, 0)
	require.Error(t, err)

	got := touches(fake.reports)
	require.NotEmpty(t, got)
	last := got[len(got)-1]
	assert.Equal(t, TouchRelease, last.state, "a broken stroke still has to lift the contact")
	assert.Equal(t, [2]uint16{20, 20}, [2]uint16{last.x, last.y},
		"released at the last point actually sent, not the end of the path")
}

func TestDragSendsOneSampleMoreThanItsSteps(t *testing.T) {
	session, fake := openSession()
	require.NoError(t, session.Drag(context.Background(), Point{X: 0, Y: 0}, Point{X: 100, Y: 0}, 4, 0))

	got := touches(fake.reports)
	// Four steps means five contacts, the first being the touch-down UIKit
	// hit-tests, plus one release.
	require.Len(t, got, 6)
	assert.Equal(t, TouchRelease, got[5].state)
	assert.Equal(t, uint16(0), got[0].x, "the first sample is the touch-down at from")
	assert.Equal(t, uint16(100), got[4].x, "the last contact reaches to")
}

func TestCloseLiftsAContactLeftDown(t *testing.T) {
	session, fake := openSession()
	require.NoError(t, session.TouchDown(context.Background(), Point{X: 42, Y: 43}))

	before := len(touches(fake.reports))
	require.NoError(t, session.Close())

	got := touches(fake.reports)
	require.Greater(t, len(got), before, "closing has to lift the held contact")
	last := got[len(got)-1]
	assert.Equal(t, TouchRelease, last.state)
	assert.Equal(t, [2]uint16{42, 43}, [2]uint16{last.x, last.y}, "lifted where the finger was")
}

func TestTouchUpWithNothingDownSendsNothing(t *testing.T) {
	session, fake := openSession()
	require.NoError(t, session.TouchUp(context.Background(), Point{X: 1, Y: 1}))
	assert.Empty(t, touches(fake.reports), "a stray release must not desynchronise the device")
}

func TestTouchMoveKeepsTheContactDown(t *testing.T) {
	session, fake := openSession()
	require.NoError(t, session.TouchDown(context.Background(), Point{X: 1, Y: 1}))
	require.NoError(t, session.TouchMove(context.Background(), Point{X: 2, Y: 2}))
	require.NoError(t, session.TouchUp(context.Background(), Point{X: 2, Y: 2}))

	got := touches(fake.reports)
	require.Len(t, got, 3)
	assert.Equal(t, TouchContact, got[0].state)
	assert.Equal(t, TouchContact, got[1].state, "a move is another contact, not a transition")
	assert.Equal(t, TouchRelease, got[2].state)
}

func TestStrokeSpreadsItsDurationAcrossThePath(t *testing.T) {
	session, _ := openSession()
	start := time.Now()
	require.NoError(t, session.Stroke(context.Background(), []Point{
		{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2},
	}, 100*time.Millisecond))
	assert.GreaterOrEqual(t, time.Since(start), 90*time.Millisecond,
		"the timestamps are what the device reads velocity from")
}

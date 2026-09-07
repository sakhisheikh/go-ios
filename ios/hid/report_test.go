package hid

import (
	"encoding/hex"
	"testing"
)

// goldenTimestamp is an arbitrary 48-bit value with distinct bytes, so a
// mis-ordered or mis-sized timestamp field shows up immediately.
const goldenTimestamp uint64 = 0x123456789ABC

// The expected byte strings below were produced by the reference implementation
// in pymobiledevice3 (remote/core_device/hid_service.py) at goldenTimestamp, so
// these tests pin this port to the wire format the device is known to accept
// rather than to our own reading of it.

func TestBuildDigitizerReport(t *testing.T) {
	tests := []struct {
		name string
		x, y int32
		want string
	}{
		{
			name: "positive coordinates",
			x:    100,
			y:    200,
			want: "1364000000c80000000000bc9a785634120000",
		},
		{
			// Coordinates are signed, so negatives must sign-extend across all
			// four bytes rather than being clamped at zero.
			name: "negative coordinates",
			x:    -1,
			y:    -2,
			want: "13fffffffffeffffff0000bc9a785634120000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDigitizerReport(tt.x, tt.y, goldenTimestamp)
			if len(got) != DigitizerReportLen {
				t.Errorf("length = %d, want %d", len(got), DigitizerReportLen)
			}
			if hex.EncodeToString(got) != tt.want {
				t.Errorf("report mismatch\n got %s\nwant %s", hex.EncodeToString(got), tt.want)
			}
		})
	}
}

func TestBuildTouchscreenReport(t *testing.T) {
	tests := []struct {
		name  string
		state TouchState
		x, y  uint16
		want  string
	}{
		{
			name:  "contact",
			state: TouchContact,
			x:     500,
			y:     1000,
			want: "090105c2f401e803000000000000000000000000000000000000000000000000" +
				"000000000000000002000000bc9a785634120000000000000000",
		},
		{
			name:  "release",
			state: TouchRelease,
			x:     500,
			y:     1000,
			want: "09010502f401e803000000000000000000000000000000000000000000000000" +
				"000000000000000002000000bc9a785634120000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildTouchscreenReport(tt.state, tt.x, tt.y, goldenTimestamp)
			if len(got) != TouchscreenReportLen {
				t.Errorf("length = %d, want %d", len(got), TouchscreenReportLen)
			}
			if hex.EncodeToString(got) != tt.want {
				t.Errorf("report mismatch\n got %s\nwant %s", hex.EncodeToString(got), tt.want)
			}
		})
	}
}

// The timestamp field is six bytes wide, so a value that does not fit has to be
// truncated to its low 48 bits instead of corrupting the trailing reserved bytes.
func TestTimestampIsTruncatedToFieldWidth(t *testing.T) {
	got := BuildDigitizerReport(0, 0, 0xFFFFFFFFFFFFFFFF)
	want := "1300000000000000000000ffffffffffff0000"
	if hex.EncodeToString(got) != want {
		t.Errorf("oversized timestamp not masked\n got %s\nwant %s", hex.EncodeToString(got), want)
	}
	if len(got) != DigitizerReportLen {
		t.Errorf("length = %d, want %d", len(got), DigitizerReportLen)
	}
}

func TestTimestampFitsInFieldAndAdvances(t *testing.T) {
	first := Timestamp()
	if first >= 1<<timestampBits {
		t.Errorf("Timestamp() = %d, wider than the %d-bit field", first, timestampBits)
	}
	second := Timestamp()
	if second < first {
		t.Errorf("Timestamp() went backwards: %d then %d", first, second)
	}
}

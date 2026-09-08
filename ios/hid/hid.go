// Package hid injects touch events over CoreDevice. Touch needs a media stream
// running to be accepted, which Session owns. iOS 27+.
package hid

import (
	"fmt"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/xpc"
)

const (
	universalServiceName = "com.apple.coredevice.hid.universalhidservice"

	universalFeatureIdentifier = "com.apple.coredevice.feature.remote.universalhidservice"
)

// surfaceMainTouchscreen is the service id the device gives its built in
// touchscreen. Every touch report goes there.
const surfaceMainTouchscreen uint64 = 257 // 0x101

type UniversalConnection struct {
	conn *xpc.Connection
}

// NewUniversal connects to the universalhidservice on the device. iOS 17+ only,
// and the Developer Disk Image must be mounted so dtuhidd is running.
func NewUniversal(device ios.DeviceEntry) (*UniversalConnection, error) {
	conn, err := ios.ConnectToXpcServiceTunnelIface(device, universalServiceName)
	if err != nil {
		return nil, fmt.Errorf("NewUniversal: %w", err)
	}
	return &UniversalConnection{conn: conn}, nil
}

// The device never returns a response, so a nil error means it was sent, not that anything moved.
func (c *UniversalConnection) sendReport(serviceID uint64, report []byte) error {
	if len(report) == 0 {
		return fmt.Errorf("sendReport: report is empty")
	}
	if err := c.conn.Send(buildSendReportPayload(serviceID, report), xpc.HeartbeatRequestFlag); err != nil {
		return fmt.Errorf("sendReport: failed to send report to surface %d: %w", serviceID, err)
	}
	return nil
}

// SendTouch posts one touch report at p. Every TouchContact means "in contact
// here", so a drag is a run of them ending in TouchRelease.
func (c *UniversalConnection) SendTouch(state TouchState, p Point) error {
	report := buildTouchscreenReport(state, p.X, p.Y, timestamp())
	if err := c.sendReport(surfaceMainTouchscreen, report); err != nil {
		return fmt.Errorf("SendTouch: %w", err)
	}
	return nil
}

func (c *UniversalConnection) Close() error {
	return c.conn.Close()
}

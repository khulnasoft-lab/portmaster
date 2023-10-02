//go:build !windows && !linux
// +build !windows,!linux

package state

import (
	"time"

	"github.com/khulnasoft-lab/portmaster/network/socket"
	"github.com/safing/portbase/config"
)

var (
	lookupTries     = 20 // With a max wait of 5ms, this amounts to up to 100ms.
	fastLookupTries = 2
)

func init() {
	// This increases performance on unsupported system.
	// It's not critical at all and does not break anything if it fails.
	go func() {
		// Wait for one minute before we set the default value, as we
		// currently cannot easily integrate into the startup procedure.
		time.Sleep(1 * time.Minute)

		// We cannot use process.CfgOptionEnableProcessDetectionKey, because of an import loop.
		config.SetDefaultConfigOption("core/enableProcessDetection", false)
	}()
}

func getTCP4Table() (connections []*socket.ConnectionInfo, listeners []*socket.BindInfo, err error) {
	return nil, nil, nil
}

func getTCP6Table() (connections []*socket.ConnectionInfo, listeners []*socket.BindInfo, err error) {
	return nil, nil, nil
}

func getUDP4Table() (binds []*socket.BindInfo, err error) {
	return nil, nil
}

func getUDP6Table() (binds []*socket.BindInfo, err error) {
	return nil, nil
}

// CheckPID checks the if socket info already has a PID and if not, tries to find it.
// Depending on the OS, this might be a no-op.
func CheckPID(socketInfo socket.Info, connInbound bool) (pid int, inbound bool, err error) {
	return socketInfo.GetPID(), connInbound, nil
}

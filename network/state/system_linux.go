package state

import (
	"time"

	"github.com/khulnasoft-lab/portmaster/network/proc"
	"github.com/khulnasoft-lab/portmaster/network/socket"
)

var (
	getTCP4Table = proc.GetTCP4Table
	getTCP6Table = proc.GetTCP6Table
	getUDP4Table = proc.GetUDP4Table
	getUDP6Table = proc.GetUDP6Table

	lookupTries     = 20 // With a max wait of 5ms, this amounts to up to 100ms.
	fastLookupTries = 2

	baseWaitTime = 3 * time.Millisecond
)

// CheckPID checks the if socket info already has a PID and if not, tries to find it.
// Depending on the OS, this might be a no-op.
func CheckPID(socketInfo socket.Info, connInbound bool) (pid int, inbound bool, err error) {
	for i := 1; i <= lookupTries; i++ {
		// look for PID
		pid = proc.GetPID(socketInfo)
		if pid != socket.UndefinedProcessID {
			// if we found a PID, return
			break
		}

		// every time, except for the last iteration
		if i < lookupTries {
			// we found no PID, we could have been too fast, give the kernel some time to think
			// back off timer: with 3ms baseWaitTime: 3, 6, 9, 12, 15, 18, 21ms - 84ms in total
			time.Sleep(time.Duration(i+1) * baseWaitTime)
		}
	}

	return pid, connInbound, nil
}

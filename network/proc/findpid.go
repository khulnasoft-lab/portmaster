//go:build linux

package proc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/khulnasoft-lab/portmaster/network/socket"
	"github.com/safing/portbase/log"
)

var (
	baseWaitTime  = 3 * time.Millisecond
	lookupRetries = 3
)

// GetPID returns the already existing pid of the given socket info or searches for it.
// This also acts as a getter for socket.*Info.PID, as locking for that occurs here.
func GetPID(socketInfo socket.Info) (pid int) {
	// Get currently assigned PID to the socket info.
	currentPid := socketInfo.GetPID()

	// If the current PID already is valid (ie. not unidentified), return it immediately.
	if currentPid != socket.UndefinedProcessID {
		return currentPid
	}

	// Find PID for the given UID and inode.
	pid = findPID(socketInfo.GetUIDandInode())

	// Set the newly found PID on the socket info.
	socketInfo.SetPID(pid)

	// Return found PID.
	return pid
}

// findPID returns the pid of the given uid and socket inode.
func findPID(uid, inode int) (pid int) {
	socketName := fmt.Sprintf("socket:[%d]", inode)

	for i := 0; i <= lookupRetries; i++ {
		var pidsUpdated bool

		// Get all pids for the given uid.
		pids, ok := getPidsByUser(uid)
		if !ok {
			// If we cannot find the uid in the map, update it.
			updatePids()
			pidsUpdated = true
			pids, ok = getPidsByUser(uid)
		}

		// If we have found PIDs, search them.
		if ok {
			// Look through the PIDs in reverse order, because higher/newer PIDs will be more likely to
			// be searched for.
			for i := len(pids) - 1; i >= 0; i-- {
				if findSocketFromPid(pids[i], socketName) {
					return pids[i]
				}
			}
		}

		// If we still cannot find our socket, and haven't yet updated the PID map,
		// do this and then check again immediately.
		if !pidsUpdated {
			updatePids()
			pids, ok = getPidsByUser(uid)
			if ok {
				// Look through the PIDs in reverse order, because higher/newer PIDs will be more likely to
				// be searched for.
				for i := len(pids) - 1; i >= 0; i-- {
					if findSocketFromPid(pids[i], socketName) {
						return pids[i]
					}
				}
			}
		}

		// We have updated the PID map, but still cannot find anything.
		// So, there is nothing we can other than wait a little for the kernel to
		// populate the information.

		// Wait after each try, except for the last iteration
		if i < lookupRetries {
			// Wait in back-off fashion - with 3ms baseWaitTime: 3, 6, 9 - 18ms in total.
			time.Sleep(time.Duration(i+1) * baseWaitTime)
		}
	}

	return socket.UndefinedProcessID
}

func findSocketFromPid(pid int, socketName string) bool {
	entries := readDirNames(fmt.Sprintf("/proc/%d/fd", pid))
	if len(entries) == 0 {
		return false
	}

	for _, entry := range entries {
		link, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, entry))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				log.Warningf("proc: failed to read link /proc/%d/fd/%s: %s", pid, entry, err)
			}
			continue
		}
		if link == socketName {
			return true
		}
	}

	return false
}

// readDirNames only reads the directory names. Using os.ReadDir() would call `lstat` on every
// resulting directory name, which we don't need. This function will be called a lot, so we should
// refrain from unnecessary work.
func readDirNames(dir string) (names []string) {
	file, err := os.Open(dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Warningf("proc: could not open directory %s: %s", dir, err)
		}
		return
	}
	defer func() {
		_ = file.Close()
	}()

	names, err = file.Readdirnames(0)
	if err != nil {
		log.Warningf("proc: could not get entries from directory %s: %s", dir, err)
		return []string{}
	}
	return
}

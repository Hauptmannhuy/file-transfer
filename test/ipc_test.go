package test

import (
	"file-transfer/bridge"
	"file-transfer/ipc"
	"fmt"
	"testing"
)

func TestCreateSharedMem(t *testing.T) {
	t.Run("create shared mem test", func(t *testing.T) {
		ipcHandler, err := ipc.InitIPC(bridge.Bridge{})
		if err != nil {
			t.Error(err)
		}

		ipcHandler.MemoryBlock[500] = 'a'
		fmt.Println("here")
		if ipcHandler.MemoryBlock[500] != 'a' {
			t.Errorf("error memory block")
		}

		ipc.UnlinkShmMem()
	})
}

package notify_lock_session

import (
	"fmt"
	"github.com/jthmath/winapi"
	"testing"
	"time"
)

var procSendMessage = user32.MustFindProc("SendMessageW")

func TestSubscribe(t *testing.T) {
	msg := make(chan Lock, 1)
	end := make(chan bool, 1)

	err := Subscribe(msg, end)
	if err != nil {
		t.Error(err)
	}
	lock := false
	go func() {
		for {
			select {
			case <-end:
				return
			case m := <-msg:
				lock = m.Lock
				fmt.Println("Is lock:", lock)
				close(end)
				return
			case <-time.After(time.Second * 20):
				t.Error("Time over")
				close(end)
				return
			}
		}
	}()

	<-time.After(time.Second * 1)

	res := SendMessage(hwnd, WM_WTSSESSION_CHANGE, WTS_SESSION_LOCK, 0)
	fmt.Println("Result:", res, hwnd)
	//_ = SendMessage(hwnd, WM_WTSSESSION_CHANGE, WTS_SESSION_UNLOCK, 0)
	//_ = SendMessage(hwnd, WM_WTSSESSION_CHANGE, WTS_SESSION_LOCK, 0)
	//	_ = SendMessage(hwnd, WM_WTSSESSION_CHANGE, WTS_SESSION_UNLOCK, 0)

	<-end
}

func SendMessage(hwnd winapi.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procSendMessage.Call(
		uintptr(hwnd),
		uintptr(msg),
		wParam,
		lParam)

	return ret
}

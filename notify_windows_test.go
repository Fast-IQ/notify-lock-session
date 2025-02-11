package notify_lock_session

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var procSendMessage = user32.MustFindProc("SendMessageW")

func TestSubscribe(t *testing.T) {
	msg := make(chan Lock, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nl := NotifyLock{}
	err := nl.Subscribe(ctx, msg)
	if err != nil {
		t.Error(err)
	}
	lock := false
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-msg:
				lock = m.Lock
				fmt.Println("Is lock:", lock)
				cancel()
				return
			case <-time.After(time.Second * 20):
				t.Error("Time over")
				cancel()
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

	<-ctx.Done()
}

func TestRemote(t *testing.T) {
	_, err := IsRemoteSession()
	if err != nil {
		t.Error(err)
	}
}

func SendMessage(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procSendMessage.Call(
		uintptr(hwnd),
		uintptr(msg),
		wParam,
		lParam)

	return ret
}

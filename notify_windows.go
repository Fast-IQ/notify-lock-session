//go:build windows

package notify_lock_session

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

func relayMessage(message uint32, wParam uintptr) {

	msg := Message{
		UMsg:  int(message),
		Param: int(wParam),
	}
	msg.ChanOk = make(chan int)

	chanMessages <- msg

	<-msg.ChanOk
}

func Subscribe(lock chan Lock, closeChan chan bool) error {
	var threadHandle HANDLE

	go func() {
		<-time.After(time.Second * 1)
		status, _ := CheckSessionStatus()
		lock <- Lock{
			Lock:  status,
			Clock: time.Now(),
		}
	}()
	go func() {
		for {
			select {
			case <-closeChan:
				log.Println("End watch lock")
				Stop(threadHandle)
				var result bool
				r0, _, msg := procCloseHandle.Call(uintptr(threadHandle), uintptr(unsafe.Pointer(&result)))
				if r0 != 0 {
					log.Printf("CloseHandle %v\n", msg)
				}
				return
			case m := <-chanMessages:
				switch m.UMsg {
				case WM_WTSSESSION_CHANGE:
					switch m.Param {
					case WTS_CONSOLE_DISCONNECT:
						l := Lock{
							Lock:  true,
							Clock: time.Now(),
						}
						lock <- l
					case WTS_CONSOLE_CONNECT:
						l := Lock{
							Lock:  false,
							Clock: time.Now(),
						}
						lock <- l
					case WTS_REMOTE_DISCONNECT:
						l := Lock{
							Lock:  true,
							Clock: time.Now(),
						}
						lock <- l
					case WTS_REMOTE_CONNECT:
						l := Lock{
							Lock:  false,
							Clock: time.Now(),
						}
						lock <- l
					case WTS_SESSION_LOCK:
						l := Lock{
							Lock:  true,
							Clock: time.Now(),
						}
						lock <- l
					case WTS_SESSION_UNLOCK:
						l := Lock{
							Lock:  false,
							Clock: time.Now(),
						}
						lock <- l
					}

				case WM_QUERYENDSESSION:
					log.Println("log off or shutdown")
				}
				close(m.ChanOk)
			}
		}
	}()

	go func() {
		var err error
		threadHandle, err = Start()
		if err != nil {
			log.Printf("CreateThread %v\n", err.Error())
		}
	}()

	return nil
}

func Start() (HANDLE, error) {
	f := WatchSessionNotifications()
	h, _, err := CreateThread(uintptr(unsafe.Pointer(&f)))

	return h, err
}

func Stop(hwnd HANDLE) {
	r0, _, err0 := syscall.SyscallN(procTerminateThread.Addr(), 0, 0, uintptr(hwnd), 0, 0, 0)
	err := int32(r0)
	if err != 0 {
		log.Printf("TerminateThread %v\n", err0.Error())
	}
}

func WatchSessionNotifications() uintptr {
	const lpClassName = "classWatchSessionNotifications"

	wc := WNDCLASS{
		PfnWndProc:   WndProc,
		PszClassName: lpClassName,
		Menu:         lpClassName,
	}
	_, err := RegisterClass(&wc)
	if err != nil {
		log.Println("Error RegisterClass:", err)
	}

	hwnd, err = CreateWindow(lpClassName,
		lpClassName,
		WS_OVERLAPPEDWINDOW,
		0,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		100, 100,
		0, 0, 0, 0)
	if err != nil {
		log.Println("Error CreateWindow:", err)
	}
	log.Println("Handle:", hwnd)
	err = UpdateWindow(hwnd)
	if err != nil {
		log.Println("Error UpdateWindow:", err)
	}

	r0, _, err0 := procWTSRegisterSessionNotification.Call(uintptr(hwnd), NOTIFY_FOR_THIS_SESSION)
	if r0 == 0 {
		log.Println("Message WTSRegisterSessionNotification:", err0)
	}

	msg := MSG{}
	res := GetMessage(&msg, 0, 0, 0)
	for res > 0 {
		_ = TranslateMessage(&msg)
		DispatchMessage(&msg)
		res = GetMessage(&msg, 0, 0, 0)
	}

	return 0
}

func WndProc(hWnd HWND, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch message {
	case WM_QUERYENDSESSION:
		relayMessage(message, lParam)
		break
	case WM_WTSSESSION_CHANGE:
		relayMessage(message, wParam)
		break
	default:
		return DefWindowProc(hWnd, message, wParam, lParam)
	}
	return 0
}

func CreateThread(proc uintptr) (h HANDLE, tid uintptr, err error) {
	lpParameter := 1
	//	r0, e1, err := syscall.SyscallN(procCreateThread.Addr(), 0, 0, proc, uintptr(unsafe.Pointer(&lpParameter)), 0, uintptr(unsafe.Pointer(&tid)))
	r0, e1, err := procCreateThread.Call(0, 0, proc, uintptr(unsafe.Pointer(&lpParameter)), 0, uintptr(unsafe.Pointer(&tid)))
	if e1 != 0 {
		//err = error(e1)
	} else {
		h = HANDLE(r0)
	}
	return h, e1, err
}

func CheckSessionStatus() (isLock bool, err error) {

	sessionId := getSessionId()
	log.Println("Session Id:", sessionId)
	/*	rs, _ := isRemoteSession(sessionId)
		log.Println("Remote session:", rs)*/
	lock, err := getLockSession(sessionId)

	return lock, err
}

func getLockSession(sessionId uint32) (isLock bool, err error) {
	var buffer *uint32
	var bytesReturned uint32

	// Получаем состояние сессии
	r1, _, _ := procWTSQuerySessionInformation.Call(
		0, // hServer (0 для локальной машины)
		uintptr(sessionId),
		uintptr(WTSSessionInfoEx),
		uintptr(unsafe.Pointer(&buffer)),
		uintptr(unsafe.Pointer(&bytesReturned)),
	)

	if r1 == 0 {
		log.Println("Error getting the session status.")
		return false, errors.New("Error getting the session status.")
	}
	b := (*WTSINFOEXA)(unsafe.Pointer(buffer))
	// состояние сессии
	switch b.Data.SessionFlags {
	case WTS_SESSIONSTATE_UNLOCK:
		return false, nil
	case WTS_SESSIONSTATE_LOCK:
		return true, nil
	default:
		return false, errors.New(strconv.Itoa(int(b.Data.SessionFlags)))
	}
}

func IsRemoteSession() (bool, error) {
	var buffer *uint32
	var bytesReturned uint32

	sessionId := getSessionId()

	// Получаем состояние сессии
	r1, _, _ := procWTSQuerySessionInformation.Call(
		0, // hServer (0 для локальной машины)
		uintptr(sessionId),
		WTSIsRemoteSession,
		uintptr(unsafe.Pointer(&buffer)),
		uintptr(unsafe.Pointer(&bytesReturned)),
	)

	if r1 == 0 {
		fmt.Println("Ошибка получения состояния сессии.")
		return false, errors.New("Ошибка получения состояния сессии. WTSIsRemoteSession")
	}
	b := (*bool)(unsafe.Pointer(buffer))
	//_, _, _ = procWTSFreeMemory.Call(uintptr(unsafe.Pointer(&buffer)))
	return *b, nil
}

func getSessionId() uint32 {
	ret, _, _ := procWTSGetActiveConsoleSessionId.Call()
	return uint32(ret)
}

type Message struct {
	UMsg   int
	Param  int
	ChanOk chan int
}

//go:build windows

package notify_lock_session

import (
	"errors"
	"github.com/jthmath/winapi"
	"log"
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
	var threadHandle winapi.HANDLE

	go func() {
		status, _ := CheckSessionStatus()
		lock <- Lock{
			Lock:  status,
			Clock: time.Now(),
		}
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

func Start() (winapi.HANDLE, error) {
	f := WatchSessionNotifications()
	h, _, err := CreateThread(uintptr(unsafe.Pointer(&f)))

	return h, err
}

func Stop(hwnd winapi.HANDLE) {
	r0, _, err0 := syscall.SyscallN(procTerminateThread.Addr(), 0, 0, uintptr(hwnd), 0, 0, 0)
	err := int32(r0)
	if err != 0 {
		log.Printf("TerminateThread %v\n", err0.Error())
	}
}

func WatchSessionNotifications() uintptr {
	const lpClassName = "classWatchSessionNotifications"

	wc := winapi.WNDCLASS{
		PfnWndProc:   WndProc,
		PszClassName: lpClassName,
		Menu:         lpClassName,
	}
	_, err := winapi.RegisterClass(&wc)
	if err != nil {
		log.Println("Error RegisterClass:", err)
	}

	hwnd, err = winapi.CreateWindow(lpClassName,
		lpClassName,
		winapi.WS_OVERLAPPEDWINDOW,
		0,
		winapi.CW_USEDEFAULT,
		winapi.CW_USEDEFAULT,
		100, 100,
		0, 0, 0, 0)
	if err != nil {
		log.Println("Error CreateWindow:", err)
	}
	log.Println("Handle:", hwnd)
	err = winapi.UpdateWindow(hwnd)
	if err != nil {
		log.Println("Error UpdateWindow:", err)
	}

	r0, _, err0 := procWTSRegisterSessionNotification.Call(uintptr(hwnd), NOTIFY_FOR_THIS_SESSION)
	if r0 != 0 {
		log.Println("Message WTSRegisterSessionNotification:", err0)
	}

	msg := winapi.MSG{}
	res := winapi.GetMessage(&msg, 0, 0, 0)
	for res > 0 {
		_ = winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
		res = winapi.GetMessage(&msg, 0, 0, 0)
	}

	return 0
}

func WndProc(hWnd winapi.HWND, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch message {
	case WM_QUERYENDSESSION:
		relayMessage(message, lParam)
		break
	case WM_WTSSESSION_CHANGE:
		relayMessage(message, wParam)
		break
	default:
		return winapi.DefWindowProc(hWnd, message, wParam, lParam)
	}
	return 0
}

func CreateThread(proc uintptr) (h winapi.HANDLE, tid uintptr, err error) {
	lpParameter := 1
	r0, e1, err := syscall.SyscallN(procCreateThread.Addr(), 0, 0, proc, uintptr(unsafe.Pointer(&lpParameter)), 0, uintptr(unsafe.Pointer(&tid)))
	if e1 != 0 {
		//err = error(e1)
	} else {
		h = winapi.HANDLE(r0)
	}
	return h, e1, err
}

func CheckSessionStatus() (isLock bool, err error) {

	r0, _, _ := procGetCurrentProcessId.Call()
	pid := uint32(r0)
	var dwSessionId uint32
	err = ProcessIdToSessionId(pid, &dwSessionId)

	if err != nil {
		return true, err
	}
	if dwSessionId == 1 {
		return false, nil
	}

	return true, nil

}

func ProcessIdToSessionId(pid uint32, sessionid *uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procProcessIdToSessionId.Addr(), uintptr(pid), uintptr(unsafe.Pointer(sessionid)))
	if r1 == 0 {
		err = errors.New("ProcessIdToSessionId error: " + e1.Error())
	}
	return
}

type Message struct {
	UMsg   int
	Param  int
	ChanOk chan int
}

var (
	hwnd winapi.HWND

	chanMessages = make(chan Message, 1000)

	kernel32 = syscall.MustLoadDLL("kernel32.dll")
	wtsapi32 = syscall.MustLoadDLL("wtsapi32.dll")
	user32   = syscall.MustLoadDLL("user32.dll")

	procWTSRegisterSessionNotification = wtsapi32.MustFindProc("WTSRegisterSessionNotification")
	procCreateThread                   = kernel32.MustFindProc("CreateThread")
	procTerminateThread                = kernel32.MustFindProc("TerminateThread")
	procCloseHandle                    = kernel32.MustFindProc("CloseHandle")
	procProcessIdToSessionId           = kernel32.MustFindProc("ProcessIdToSessionId")
	procGetCurrentProcessId            = kernel32.MustFindProc("GetCurrentProcessId")
)

// http://msdn.microsoft.com/en-us/library/aa383828(v=vs.85).aspx
const (
	WTS_CONSOLE_CONNECT        = 0x1
	WTS_CONSOLE_DISCONNECT     = 0x2
	WTS_REMOTE_CONNECT         = 0x3
	WTS_REMOTE_DISCONNECT      = 0x4
	WTS_SESSION_LOGON          = 0x5
	WTS_SESSION_LOGOFF         = 0x6
	WTS_SESSION_LOCK           = 0x7
	WTS_SESSION_UNLOCK         = 0x8
	WTS_SESSION_REMOTE_CONTROL = 0x9
	WTS_SESSION_CREATE         = 0xA
	WTS_SESSION_TERMINATE      = 0xB

	WM_QUERYENDSESSION   = 0x11
	WM_WTSSESSION_CHANGE = 0x2B1

	ENDSESSION_CLOSEAPP = 0x00000001
	ENDSESSION_CRITICAL = 0x40000000
	ENDSESSION_LOGOFF   = 0x80000000
)

const (
	NOTIFY_FOR_THIS_SESSION = 0
	NOTIFY_FOR_ALL_SESSIONS = 1
)

const (
	PROC_TOKEN_DUPLICATE         = 0x0002
	PROC_TOKEN_QUERY             = 0x0008
	PROC_TOKEN_ADJUST_PRIVILEGES = 0x0020
)

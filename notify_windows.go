//go:build windows

package notify_lock_session

import (
	"errors"
	"fmt"
	"github.com/jthmath/winapi"
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

	sessionId := getSessionId()
	fmt.Println("Session Id:", sessionId)
	rs, _ := isRemoteSession(sessionId)
	fmt.Println("Remote session:", rs)
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
		fmt.Println("Ошибка получения состояния сессии.")
		return false, errors.New("error")
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

func isRemoteSession(sessionId uint32) (bool, error) {
	var buffer *uint32
	var bytesReturned uint32

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
	_, _, _ = procWTSFreeMemory.Call(uintptr(unsafe.Pointer(&buffer)))
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

var (
	hwnd winapi.HWND

	chanMessages = make(chan Message, 1000)

	kernel32 = syscall.MustLoadDLL("kernel32.dll")
	wtsapi32 = syscall.MustLoadDLL("wtsapi32.dll")
	user32   = syscall.MustLoadDLL("user32.dll")

	procWTSRegisterSessionNotification = wtsapi32.MustFindProc("WTSRegisterSessionNotification")
	procWTSQuerySessionInformation     = wtsapi32.MustFindProc("WTSQuerySessionInformationW")
	procWTSFreeMemory                  = wtsapi32.MustFindProc("WTSFreeMemory")
	procCreateThread                   = kernel32.MustFindProc("CreateThread")
	procTerminateThread                = kernel32.MustFindProc("TerminateThread")
	procCloseHandle                    = kernel32.MustFindProc("CloseHandle")
	//	procProcessIdToSessionId           = kernel32.MustFindProc("ProcessIdToSessionId")
	//	procGetCurrentProcessId            = kernel32.MustFindProc("GetCurrentProcessId")
	procWTSGetActiveConsoleSessionId = kernel32.MustFindProc("WTSGetActiveConsoleSessionId")
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

const (
	WTS_SESSIONSTATE_LOCK    = 0x0
	WTS_SESSIONSTATE_UNLOCK  = 0x1
	WTS_SESSIONSTATE_UNKNOWN = 0xFFFFFFFF
)

const (
	WTSInitialProgram     = 0
	WTSApplicationName    = 1
	WTSWorkingDirectory   = 2
	WTSOEMId              = 3
	WTSSessionId          = 4
	WTSUserName           = 5
	WTSWinStationName     = 6
	WTSDomainName         = 7
	WTSConnectState       = 8
	WTSClientBuildNumber  = 9
	WTSClientName         = 10
	WTSClientDirectory    = 11
	WTSClientProductId    = 12
	WTSClientHardwareId   = 13
	WTSClientAddress      = 14
	WTSClientDisplay      = 15
	WTSClientProtocolType = 16
	WTSIdleTime           = 17
	WTSLogonTime          = 18
	WTSIncomingBytes      = 19
	WTSOutgoingBytes      = 20
	WTSIncomingFrames     = 21
	WTSOutgoingFrames     = 22
	WTSClientInfo         = 23
	WTSSessionInfo        = 24
	WTSSessionInfoEx      = 25
	WTSConfigInfo         = 26
	WTSValidationInfo     = 27
	WTSSessionAddressV4   = 28
	WTSIsRemoteSession    = 29
)

const (
	WINSTATIONNAME_LENGTH = 32
	USERNAME_LENGTH       = 20
	DOMAIN_LENGTH         = 15
)

type WTS_CONNECTSTATE_CLASS int32

// Структура, аналогичная WTSINFOEX_LEVEL1_A в Go
type WTSINFOEX_LEVEL1_A struct {
	SessionId               uint32
	SessionState            WTS_CONNECTSTATE_CLASS
	SessionFlags            int32
	WinStationName          [WINSTATIONNAME_LENGTH + 1]uint16
	UserName                [USERNAME_LENGTH + 1]uint16
	DomainName              [DOMAIN_LENGTH + 1]uint16
	LogonTime               time.Time
	ConnectTime             time.Time
	DisconnectTime          time.Time
	LastInputTime           time.Time
	CurrentTime             int64
	IncomingBytes           uint32
	OutgoingBytes           uint32
	IncomingFrames          uint32
	OutgoingFrames          uint32
	IncomingCompressedBytes uint32
	OutgoingCompressedBytes uint32
}

// Структура WTSINFOEXA
type WTSINFOEXA struct {
	Level uint32
	Data  WTSINFOEX_LEVEL1_A
}

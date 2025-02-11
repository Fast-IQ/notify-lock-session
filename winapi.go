//go:build windows

package notify_lock_session

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
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

const (
	FORMAT_MESSAGE_IGNORE_INSERTS = 0x00000200
	FORMAT_MESSAGE_FROM_STRING    = 0x00000400
	FORMAT_MESSAGE_FROM_HMODULE   = 0x00000800
	FORMAT_MESSAGE_FROM_SYSTEM    = 0x00001000
	FORMAT_MESSAGE_ARGUMENT_ARRAY = 0x00002000
	FORMAT_MESSAGE_MAX_WIDTH_MASK = 0x000000FF
)

/*
 * Window Styles
 */
const (
	WS_OVERLAPPED   uint32 = 0x00000000
	WS_POPUP        uint32 = 0x80000000
	WS_CHILD        uint32 = 0x40000000
	WS_MINIMIZE     uint32 = 0x20000000
	WS_VISIBLE      uint32 = 0x10000000
	WS_DISABLED     uint32 = 0x08000000
	WS_CLIPSIBLINGS uint32 = 0x04000000
	WS_CLIPCHILDREN uint32 = 0x02000000
	WS_MAXIMIZE     uint32 = 0x01000000
	WS_CAPTION      uint32 = 0x00C00000 // WS_BORDER | WS_DLGFRAME
	WS_BORDER       uint32 = 0x00800000
	WS_DLGFRAME     uint32 = 0x00400000
	WS_VSCROLL      uint32 = 0x00200000
	WS_HSCROLL      uint32 = 0x00100000
	WS_SYSMENU      uint32 = 0x00080000
	WS_THICKFRAME   uint32 = 0x00040000
	WS_GROUP        uint32 = 0x00020000
	WS_TABSTOP      uint32 = 0x00010000

	WS_MINIMIZEBOX uint32 = 0x00020000
	WS_MAXIMIZEBOX uint32 = 0x00010000

	WS_TILED       uint32 = WS_OVERLAPPED
	WS_ICONIC      uint32 = WS_MINIMIZE
	WS_SIZEBOX     uint32 = WS_THICKFRAME
	WS_TILEDWINDOW uint32 = WS_OVERLAPPEDWINDOW

	/*
	 * Common Window Styles
	 */
	WS_OVERLAPPEDWINDOW uint32 = WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU |
		WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX

	WS_POPUPWINDOW uint32 = WS_POPUP | WS_BORDER | WS_SYSMENU

	WS_CHILDWINDOW uint32 = WS_CHILD
)

const CW_USEDEFAULT int32 = ^int32(0x7FFFFFFF) // 0x80000000

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
type HANDLE uintptr

type HWND uintptr

type HMENU uintptr
type HINSTANCE uintptr
type HMODULE uintptr

type HGDIOBJ uintptr
type HDC uintptr
type HICON uintptr
type HCURSOR uintptr
type HBRUSH uintptr
type HBITMAP uintptr

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDPROC func(HWND, uint32, uintptr, uintptr) uintptr

type WNDCLASS struct {
	Style         uint32
	PfnWndProc    WNDPROC
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         HICON
	HCursor       HCURSOR
	HbrBackground HBRUSH
	Menu          interface{}
	PszClassName  string
	HIconSmall    HICON
}

type _WNDCLASS struct {
	cbSize        uint32
	style         uint32
	pfnWndProcPtr uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     HINSTANCE
	hIcon         HICON
	hCursor       HCURSOR
	hbrBackground HBRUSH
	pszMenuName   *uint16
	pszClassName  *uint16
	hIconSmall    HICON
}

type WinErrorCode uint32

var (
	hwnd HWND

	chanMessages = make(chan Message, 1000)

	kernel32 = syscall.MustLoadDLL("kernel32.dll")
	wtsapi32 = syscall.MustLoadDLL("wtsapi32.dll")
	user32   = syscall.MustLoadDLL("user32.dll")

	procWTSRegisterSessionNotification = wtsapi32.MustFindProc("WTSRegisterSessionNotification")
	procWTSQuerySessionInformation     = wtsapi32.MustFindProc("WTSQuerySessionInformationW")
	procWTSFreeMemory                  = wtsapi32.MustFindProc("WTSFreeMemory")
	procWTSGetActiveConsoleSessionId   = kernel32.MustFindProc("WTSGetActiveConsoleSessionId")
	procCreateThread                   = kernel32.MustFindProc("CreateThread")
	procTerminateThread                = kernel32.MustFindProc("TerminateThread")
	procCloseHandle                    = kernel32.MustFindProc("CloseHandle")
	procFormatMessage                  = kernel32.MustFindProc("FormatMessageW")
	//	procProcessIdToSessionId           = kernel32.MustFindProc("ProcessIdToSessionId")
	//	procGetCurrentProcessId            = kernel32.MustFindProc("GetCurrentProcessId")
	procTranslateMessage = user32.MustFindProc("TranslateMessage")
	procGetMessage       = user32.MustFindProc("GetMessageW")
	procDispatchMessage  = user32.MustFindProc("DispatchMessageW")
	procDefWindowProc    = user32.MustFindProc("DefWindowProcW")
	procUpdateWindow     = user32.MustFindProc("UpdateWindow")
	procCreateWindowExW  = user32.MustFindProc("CreateWindowExW")
	procRegisterClassExW = user32.MustFindProc("RegisterClassExW")
)

func FormatMessage(flags uint32, msgsrc interface{}, msgid uint32, langid uint32, args *byte) (string, error) {
	var b [300]uint16
	n, err := _FormatMessage(flags, msgsrc, msgid, langid, &b[0], 300, args)
	if err != nil {
		return "", err
	}
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n])), nil
}

func _FormatMessage(flags uint32, msgsrc interface{}, msgid uint32, langid uint32, buf *uint16, nSize uint32, args *byte) (n uint32, err error) {
	r0, _, e1 := procFormatMessage.Call(
		uintptr(flags), uintptr(0), uintptr(msgid), uintptr(langid),
		uintptr(unsafe.Pointer(buf)), uintptr(nSize),
		uintptr(unsafe.Pointer(args)), 0, 0)
	n = uint32(r0)
	if n == 0 {
		err = fmt.Errorf("winapi._FormatMessage error: %d", e1)
	}
	return
}

func (this WinErrorCode) Error() string {
	var flags uint32 = FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_ARGUMENT_ARRAY | FORMAT_MESSAGE_IGNORE_INSERTS
	str, err := FormatMessage(flags, nil, uint32(this), 0, nil)
	n := uint32(this)
	if err == nil {
		return fmt.Sprintf("winapi error: %d(0x%08X) - ", n, n) + str
	} else {
		return fmt.Sprintf("winapi error: %d(0x%08X)", n, n)
	}
}

func GetMessage(pMsg *MSG, hWnd HWND, wMsgFilterMin uint32, wMsgFilterMax uint32) int32 {
	r1, _, _ := procGetMessage.Call(
		uintptr(unsafe.Pointer(pMsg)),
		uintptr(hWnd),
		uintptr(wMsgFilterMin),
		uintptr(wMsgFilterMax),
		0, 0)
	return int32(r1)
}

func TranslateMessage(pMsg *MSG) error {
	r1, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(pMsg)), 0, 0)
	if r1 == 0 {
		return errors.New("winapi: TranslateMessage failed.")
	} else {
		return nil
	}
}

func DispatchMessage(pMsg *MSG) uintptr {
	r1, _, _ := procDispatchMessage.Call(uintptr(unsafe.Pointer(pMsg)), 0, 0)
	return r1
}

func DefWindowProc(hWnd HWND, message uint32, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProc.Call(uintptr(hWnd), uintptr(message), wParam, lParam, 0, 0)
	return ret
}

func UpdateWindow(hWnd HWND) error {
	r1, _, _ := procUpdateWindow.Call(uintptr(hWnd), 0, 0)
	if r1 == 0 {
		return errors.New("winapi: UpdateWindow failed.") // 该函数没有对应的GetLastError值
	} else {
		return nil
	}
}

func CreateWindowExW(ClassName string, WindowName string, Style uint32, ExStyle uint32,
	X int32, Y int32, Width int32, Height int32,
	WndParent HWND, Menu HMENU, inst HINSTANCE, Param uintptr) (hWnd HWND, err error) {
	pClassName, err := syscall.UTF16PtrFromString(ClassName)
	if err != nil {
		return 0, err
	}
	pWindowName, err := syscall.UTF16PtrFromString(WindowName)
	if err != nil {
		return 0, err
	}
	r1, _, err := procCreateWindowExW.Call(
		uintptr(ExStyle), uintptr(unsafe.Pointer(pClassName)), uintptr(unsafe.Pointer(pWindowName)), uintptr(Style),
		uintptr(X), uintptr(Y), uintptr(Width), uintptr(Height),
		uintptr(WndParent), uintptr(Menu), uintptr(inst), uintptr(Param))
	if r1 == 0 {
		if err == nil {
			return 0, errors.New("winapi: CreateWindow failed. " + syscall.GetLastError().Error())
		}
	} else {
		hWnd = HWND(r1)
	}
	return hWnd, nil
}

func newWndProc(proc WNDPROC) uintptr {
	return syscall.NewCallback(proc)
}

func RegisterClass(pWndClass *WNDCLASS) (atom uint16, err error) {
	if pWndClass == nil {
		err = errors.New("winapi: RegisterClass: pWndClass must not be nil.")
		return
	}

	_pClassName, err := syscall.UTF16PtrFromString(pWndClass.PszClassName)
	if err != nil {
		return
	}

	if pWndClass.Menu == nil {
		err = errors.New("winapi: RegisterClass: can't find Menu.")
		return
	}

	var Menu uintptr = 70000

	var _pMenuName *uint16 = nil

	switch v := pWndClass.Menu.(type) {
	case uint16:
		Menu = MakeIntResource(v)
	case string:
		_pMenuName, err = syscall.UTF16PtrFromString(v)
		if err != nil {
			return
		}
	default:
		return 0, errors.New("winapi: RegisterClass: Menu's type must be uint16 or string.")
	}

	var wc _WNDCLASS
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.style = pWndClass.Style
	wc.pfnWndProcPtr = newWndProc(pWndClass.PfnWndProc)
	wc.cbClsExtra = pWndClass.CbClsExtra
	wc.cbWndExtra = pWndClass.CbWndExtra
	wc.hInstance = pWndClass.HInstance
	wc.hIcon = pWndClass.HIcon
	wc.hCursor = pWndClass.HCursor
	wc.hbrBackground = pWndClass.HbrBackground
	if _pClassName != nil {
		wc.pszMenuName = _pMenuName
	} else {
		wc.pszMenuName = (*uint16)(unsafe.Pointer(Menu))
	}
	wc.pszClassName = _pClassName
	wc.hIconSmall = pWndClass.HIconSmall

	r1, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)), 0, 0)
	n := uint16(r1)

	if n == 0 {
		err = syscall.GetLastError()
		/*wec := WinErrorCode(e1)
		if wec != 0 {
			err = wec
		} else {
			err = errors.New("winapi: RegisterClass failed.")
		}*/
	} else {
		atom = n
	}
	return
}

func MakeIntResource(id uint16) uintptr {
	return uintptr(id)
}

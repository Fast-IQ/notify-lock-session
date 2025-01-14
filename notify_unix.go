//go:build linux

package notify_lock_session

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"log"
	"os"
	"time"
)

type paramDBUS struct {
	iface  string
	member string
}

func Subscribe(lock chan Lock, closeChan chan bool) (e error) {
	go func() {
		// Подключение к системе D-Bus
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			e = err
			return
			//log.Fatalf("Don`t connect D-Bus: %v", err)
			//return err
		}
		defer func() { _ = conn.Close() }()
		param := getDbusParams()
		// Подписка на события
		err = conn.AddMatchSignal(
			dbus.WithMatchInterface(param.iface),
			dbus.WithMatchMember(param.member),
		)
		if err != nil {
			log.Fatalf("Error subscribe on event: %v", err)
			//return err
		}

		// Канал для получения сигналов
		var signals = make(chan *dbus.Signal, 10)
		conn.Signal(signals)

		for {
			select {
			case s := <-signals:
				if len(s.Body) > 0 {
					state, ok := s.Body[0].(bool)
					if ok {
						l := Lock{
							Lock:  state,
							Clock: time.Now(),
						}
						lock <- l
					}
				}
			case <-closeChan:
				return
			}

		}
	}()

	return nil
}

func getDbusParams() (p paramDBUS) {
	osDesc := os.Getenv("XDG_CURRENT_DESKTOP")
	fmt.Println(osDesc)
	switch osDesc {
	case "ubuntu:GNOME": //work tested
		p.iface = "org.gnome.ScreenSaver"
		p.member = "ActiveChanged"
	case "Unity":
		p.iface = "com.canonical.Unity"
		p.member = "ActiveChanged"
	case "KDE":
		p.iface = "org.kde.screensaver"
		p.member = "ActiveChanged" //ok
	case "XFCE":
		p.iface = "xfce4-session"
		p.member = "???"
	case "LXQt":
	case "Cinnamon":
		p.iface = "org.Cinnamon.ScreenSaver"
		p.member = "ActiveChanged" //ok
	case "LXDE":
	case "Deepin":
		p.iface = "com.deepin.ScreenSaver"
		p.member = "???"
	default: // default to gnome
		p.iface = "org.gnome.ScreenSaver"
		p.member = "ActiveChanged"
	}
	return
}

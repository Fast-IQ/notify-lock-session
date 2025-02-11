//go:build darwin

//NOT TEST!!!

package notify_lock_session

/*
#cgo LDFLAGS: -framework Foundation -framework AppKit
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

// Функция для добавления наблюдателя
void addObserver() {
    // Создаем объект NSNotificationCenter
    NSNotificationCenter *center = [NSNotificationCenter defaultCenter];

    // Добавляем наблюдателя на уведомления о блокировке экрана (система)
    [center addObserverForName:NSWorkspaceSessionDidBecomeActiveNotification
                        object:nil
                         queue:nil
                    usingBlock:^(NSNotification *note) {
						relayMessage(0)
                        NSLog(@"Сессия стала активной!");
                    }];

    // Добавляем наблюдателя на другие уведомления (например, уход в режим сна)
    [center addObserverForName:NSWorkspaceSessionDidResignActiveNotification
                        object:nil
                         queue:nil
                    usingBlock:^(NSNotification *note) {
						relayMessage(1)
                        NSLog(@"Сессия была заблокирована или завершена!");
                    }];
}

*/

import "C"
import (
	"context"
	"time"
)

var messages = make(chan bool)

//export relayMessage
func (l *NotifyLock) relayMessage(lock C.uint) {
	messages <- lock != 0
}

func (l *NotifyLock) Subscribe(ctx context.Context, lock chan Lock) error {
	go func() {
		C.addObserver()
		for {
			select {
			case m <- messages:
				l := Lock{
					Lock:  m,
					Clock: time.Now(),
				}
			case <-ctx.Done():
				return
			}
		}
	}
	return nil
}

func IsRemoteSession() (bool, error) {
	out, err := exec.Command("lsof", "-i").Output()
	if err != nil {
		return false, error.New("Ошибка при выполнении команды lsof:", err)

	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":5900") { // Порт VNC
			return true, nil
		}
		if strings.Contains(line, ":22") { // Порт SSH
			return true, nil
		}
		if strings.Contains(line, ":3283") { // Порт ARD
			return true, nil
		}
	}
	return false, nil
}

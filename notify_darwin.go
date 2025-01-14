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
import "time"

var messages = make(chan bool)

//export relayMessage
func relayMessage(lock C.uint) {
	messages <- lock != 0
}

func Subscribe(lock chan Lock, closeChan chan bool) error {
	go func() {
		C.addObserver()
		for {
			select {
			case m <- messages:
				l := Lock{
					Lock:  m,
					Clock: time.Now(),
				}
			case <-closeChan:
				return
			}
		}
	}
	return nil
}

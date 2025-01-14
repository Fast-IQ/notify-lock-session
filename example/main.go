package main

import (
	"fmt"
	notifyLS "github.com/Fast-IQ/notify-lock-session"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGABRT)

	chanClose := make(chan bool, 1)
	end := make(chan bool, 1)

	go func() {
		for {
			<-quit
			fmt.Println("Exit. Wait close.")
			chanClose <- true
			end <- true
			//other close operation
		}
	}()

	info := make(chan notifyLS.Lock, 10)

	_ = notifyLS.Subscribe(info, chanClose)

	i := 0
	for i == 0 {
		select {
		case l := <-info:
			if l.Lock {
				fmt.Println(l.Clock, "Session lock")
			} else {
				fmt.Println(l.Clock, "Session unlock")
			}
		case <-end:
			i = 1
		}
	}
}

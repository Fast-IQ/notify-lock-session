package main

import (
	"fmt"
	notifyLS "github.com/Fast-IQ/notify-lock-session"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGABRT)

	chanClose := make(chan bool, 1)
	//end := make(chan bool, 1)

	go func() {
		for {
			<-quit
			fmt.Println("Exit. Wait close.")
			close(chanClose)
			//close(end)
			//other close operation
		}
	}()

	info := make(chan notifyLS.Lock, 10)

	_ = notifyLS.Subscribe(info, chanClose)

	remote, err := notifyLS.IsRemoteSession()
	if err != nil {
		fmt.Println("Error get info of remote session: ", err)
	}

	if remote {
		fmt.Println("This is local session.")
	} else {
		fmt.Println("This is remote session.")
	}

	for {
		select {
		case l := <-info:
			if l.Lock {
				fmt.Println(l.Clock, "Session lock")
			} else {
				fmt.Println(l.Clock, "Session unlock")
			}
		case <-chanClose:
			log.Println("End loop lock")
			return
		}
	}
}

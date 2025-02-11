package main

import (
	"context"
	"fmt"
	notifyLS "github.com/Fast-IQ/notify-lock-session"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGABRT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			<-quit
			fmt.Println("Exit. Wait close.")
			cancel()
			//other close operation
		}
	}()

	info := make(chan notifyLS.Lock, 10)
	nl := notifyLS.NotifyLock{}
	_ = nl.Subscribe(ctx, info)

	remote, err := notifyLS.IsRemoteSession()
	if err != nil {
		fmt.Println("Error get info of remote session: ", err)
	}

	if remote {
		fmt.Println("This is remote session.")
	} else {
		fmt.Println("This is local session.")
	}

	for {
		select {
		case l := <-info:
			if l.Lock {
				fmt.Println(l.Clock, "Session lock")
			} else {
				fmt.Println(l.Clock, "Session unlock")
			}
		case <-ctx.Done():
			slog.Info("End loop lock")
			return
		}
	}
}

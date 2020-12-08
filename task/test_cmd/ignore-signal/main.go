package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	fmt.Println("Ready. Awaiting signals...")
	for sig := range signals {
		switch sig {
		case syscall.SIGTERM:
			fmt.Println("Nope, not gonna")

		case syscall.SIGINT:
			fmt.Println("OK, I'm going quietly")
			return
		}
	}
}

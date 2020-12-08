package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	fmt.Println("Ready. Waiting for SIGTERM...")
	<-signals
	fmt.Println("Caught SIGTERM, exiting")
}

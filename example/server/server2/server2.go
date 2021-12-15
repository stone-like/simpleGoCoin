package main

import (
	"os"
	"os/signal"
	s "simpleGoCoin"
	"syscall"
)

func main() {
	config := s.DefaultConfig()
	config.SetHost("127.0.0.1")
	config.SetPort("5002")
	cm := s.NewConnectionManager(config.Host, config.Port, config.TCPtimeout, config.PINGinterval, config.Logger)

	server := s.NewServer(config, cm)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	server.Run()
	server.Join("127.0.0.1", "5001")

	<-quit
	server.ShutDown()

}

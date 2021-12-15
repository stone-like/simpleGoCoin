package main

import (
	"os"
	"os/signal"
	s "simpleGoCoin"
	"syscall"
	"time"
)

// func main() {
// 	config := s.DefaultConfig()
// 	config.SetHost("127.0.0.1")
// 	config.SetPort("6001")
// 	cm := s.NewEdgeConnectionManager(config.Host, config.Port, config.TCPtimeout, config.PINGinterval, config.Logger)

// 	cli := s.NewClient(config, cm)

// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
// 	cli.Run()
// 	cli.Join("127.0.0.1", "5001")

// 	<-quit
// 	cli.ShutDown()
// }

func main() {
	config := s.DefaultConfig()
	config.SetHost("127.0.0.1")
	config.SetPort("6002")
	cm := s.NewEdgeConnectionManager(config.Host, config.Port, config.TCPtimeout, config.PINGinterval, config.Logger)

	cli := s.NewClient(config, cm)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	cli.Run()
	cli.Join("127.0.0.1", "5001")

	//clientNode登録を待つ
	time.Sleep(1 * time.Second)
	cli.SendMessage("Hi I am client!")

	<-quit
	cli.ShutDown()
}

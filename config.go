package simplegocoin

import (
	"log"
	"time"
)

type Config struct {
	Host         string
	Port         string
	TCPtimeout   time.Duration
	PINGinterval time.Duration
	Logger       *log.Logger
}

//defaultの出力はstderr
var DefaultLogger = log.Default()

func DefaultConfig() *Config {
	return &Config{
		TCPtimeout:   10 * time.Second,
		PINGinterval: 3 * time.Second,
		Logger:       DefaultLogger,
	}
}

func (c *Config) SetPort(port string) {
	c.Port = port
}

func (c *Config) SetHost(host string) {
	c.Host = host
}

func (c *Config) SetLogger(logger *log.Logger) {
	c.Logger = logger
}

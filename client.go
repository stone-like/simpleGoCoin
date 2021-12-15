package simplegocoin

import "log"

type ClientState int

const (
	CLIENT_STATE_INIT ClientState = iota + 1
	CLIENT_STATE_ACTIVE
	CLIENT_STATE_SHUTTING_DOWN
)

var connectionProtocol = "tcp"

type Client struct {
	host   string
	port   string
	state  ClientState
	logger *log.Logger
	cm     *EdgeConnectionManager
}

func NewClient(config *Config, cm *EdgeConnectionManager) *Client {
	return &Client{
		host:   config.Host,
		port:   config.Port,
		state:  CLIENT_STATE_INIT,
		logger: config.Logger,
		cm:     cm,
	}
}

func (c *Client) Run() {
	c.state = CLIENT_STATE_ACTIVE
	c.cm.Run()
}

func (c *Client) Join(host, port string) {
	c.cm.Join(host, port)
}

func (c *Client) ShutDown() {
	c.state = CLIENT_STATE_SHUTTING_DOWN
	c.logger.Println("Shutdon Edge Node...")
	c.cm.ShutDown()
}

func (c *Client) GetState() ClientState {
	return c.state
}

func (c *Client) SendMessage(content string) {
	c.cm.SendMessage(MSG_ENHANCED, content)
}

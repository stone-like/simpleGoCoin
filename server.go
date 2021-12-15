package simplegocoin

import "log"

type State int

const (
	STATE_INIT State = iota + 1
	STATE_STANDBY
	STATE_CONNECTED_TO_NETWORK
	STATE_SHUTTING_DOWN
)

type Server struct {
	host   string
	port   string
	state  State
	logger *log.Logger
	cm     *ConnectionManager
}

func NewServer(config *Config, cm *ConnectionManager) *Server {
	return &Server{
		host:   config.Host,
		port:   config.Port,
		state:  STATE_INIT,
		logger: config.Logger,
		cm:     cm,
	}
}

func (s *Server) Run() {
	s.state = STATE_STANDBY
	s.cm.Run()
}

func (s *Server) Join(host, port string) {
	s.state = STATE_CONNECTED_TO_NETWORK
	s.cm.Join(host, port)
}

func (s *Server) ShutDown() {
	s.state = STATE_SHUTTING_DOWN
	s.logger.Println("Shutdon server...")
	s.cm.ShutDown()
}

func (s *Server) GetState() State {
	return s.state
}

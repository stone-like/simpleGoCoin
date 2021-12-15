package simplegocoin

import (
	"log"
	"net"
	"strings"
	"time"
)

//Edgeの場合はCoreと違って対象のCoreサーバーとの一対一の通信のみでよくてそのほかのサーバーと情報共有の必要はない
type EdgeConnectionManager struct {
	host              string
	port              string
	coreNodeList      *CoreNodeList
	messageManager    *MessageManager
	currentTargetCore string
	shutdownCh        chan struct{}
	stopCh            chan struct{}
	streamCh          chan net.Conn
	TCPtimeout        time.Duration
	PINGinterval      time.Duration
	logger            *log.Logger
	listener          net.Listener
	tickers           []*time.Ticker
}

func NewEdgeConnectionManager(host, port string, TCPtimeout, PINGinterval time.Duration, logger *log.Logger) *EdgeConnectionManager {
	mm := NewMessageManeger()
	cm := &EdgeConnectionManager{
		host:           host,
		port:           port,
		coreNodeList:   NewCoreNodeList(logger),
		messageManager: mm,
		shutdownCh:     make(chan struct{}),
		stopCh:         make(chan struct{}),
		streamCh:       make(chan net.Conn),
		logger:         logger,
		TCPtimeout:     TCPtimeout,
		PINGinterval:   PINGinterval,
	}

	return cm
}

//4んでいるノードをsliceに追加
func (e *EdgeConnectionManager) Ping(addr string) bool {
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		e.logger.Println(err)
		return false
	}

	defer conn.Close()

	msg, err := e.messageManager.CreateWithoutPayload(e.port, MSG_PING)
	if err != nil {
		e.logger.Println(err)
		return false
	}

	conn.Write(msg)
	//将来的にpingの結果が返ってきたらtrueとする

	return true
}

//もしかしたらheartBeat自体にtimeoutやロックをかけて複数のheartBeatが同時に発火しないようにした方がいいかもしれない
func (e *EdgeConnectionManager) HeartBeat() {
	e.logger.Println("HeartBeat was Called")

	if e.currentTargetCore == "" {
		return
	}

	if ok := e.Ping(e.currentTargetCore); ok {
		return
	}
	//pingが繋がらなかったときは他のCoreNodeへ
	for node, _ := range e.coreNodeList.CurrentNodes() {
		ok := e.Ping(node)
		if ok {
			e.currentTargetCore = node
			return
		}
	}
	//繋がるCoreが一つもないときはShutDown
	e.ShutDown()

}

func (e *EdgeConnectionManager) Serve() {

	l, err := net.Listen(connectionProtocol, getAddress(e.host, e.port))
	if err != nil {
		log.Fatal(err)
		return
	}

	e.listener = l

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
			return

		}

		e.streamCh <- conn
	}
}

//待ち受けを開始する
//chを返して、Serveが終了したらそこで呼び出し元も終了させるGracefulShutDownはユーザー側の責務とする？
func (e *EdgeConnectionManager) Run() {

	//メッセージ受信用のGoroutine
	go e.Serve()
	//メッセージ処理用のGoroutine
	go e.messageHandler()
	//heartBeat用のGoRoutine
	go e.schedule()

}

//ユーザーが指定したCoreノードへ接続
func (e *EdgeConnectionManager) Join(host, port string) {
	addr := getAddress(host, port)
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		e.logger.Println(err)
		return
	}

	defer conn.Close()

	msg, err := e.messageManager.CreateWithoutPayload(e.port, MSG_ADD_AS_EDGE)
	if err != nil {
		e.logger.Println(err)
		return
	}
	e.Send(msg, addr)

	e.currentTargetCore = addr

}

func (e *EdgeConnectionManager) SendToCore(msg []byte, addr string) bool {
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		e.logger.Println(err)
		e.logger.Println("fail to connect peer")
		//本当はremoveの判定をもっと優しくしたりした方がいいはず(retryするとか、ゴシッププロトコルみたいにindirectProbeするとか)
		e.coreNodeList.Remove(addr)
		return false
	}
	defer conn.Close()

	conn.Write(msg)

	return true
}

//選択したノードに対しメッセージを送る
func (e *EdgeConnectionManager) Send(msg []byte, addr string) {

	if addr == "" {
		return
	}

	if ok := e.SendToCore(msg, addr); ok {
		return
	}

	for node, _ := range e.coreNodeList.CurrentNodes() {
		ok := e.SendToCore(msg, node)
		if ok {
			e.currentTargetCore = node
			return
		}
	}

	//繋がるCoreが一つもないときはShutDown
	e.currentTargetCore = ""
	e.ShutDown()

}
func (e *EdgeConnectionManager) Leave() {
	msg, err := e.messageManager.CreateWithoutPayload(e.port, MSG_REMOVE_EDGE)
	if err != nil {
		e.logger.Println(err)
		return
	}
	e.Send(msg, e.currentTargetCore)

	e.ShutDown()

}

//wgまでやってグレイスフルシャットダウンはいらない気がする?.memberlistとか普通の分散サーバーで一つのServerStructが複数のlistenerを立てるとかならいるのかな、でも今回p2pは1Serverにつき一人の想定なのでいらないかも
//shutDown,shutDownChはサーバー自体をシャットダウン、
//deschedule,stopChはheartbeatをstop
func (e *EdgeConnectionManager) ShutDown() {

	close(e.shutdownCh)
	e.listener.Close() //これによりAcceptが新規受付しなくなる

	e.deschedule()
}

func (e *EdgeConnectionManager) deschedule() {
	close(e.stopCh)
	//tickerをstop
	for _, t := range e.tickers {
		t.Stop()
	}
	e.tickers = nil
}

func (e *EdgeConnectionManager) triggerFunc(duration time.Duration, C <-chan time.Time, stop <-chan struct{}, f func()) {

	for {
		select {
		case <-C:
			f()
		case <-stop:
			return
		}
	}

}

func (e *EdgeConnectionManager) schedule() {
	// Create a new probeTicker
	if e.PINGinterval > 0 {
		t := time.NewTicker(e.PINGinterval)
		go e.triggerFunc(e.PINGinterval, t.C, e.stopCh, e.HeartBeat)
		e.tickers = append(e.tickers, t)
	}
}

func (e *EdgeConnectionManager) StreamCh() <-chan net.Conn {
	return e.streamCh
}

//connectionManegerをmemberListのTransport部分とした方が良さそう？
//shutdownの時transport,つまりtcpのserveを閉じる(wgを使用)、そのあとはコネクション関係ないのでshutDownChを使って閉じるだけで良さそう
//この中のhandleMessageより先にServerが終わっても、shutdownChをcloseしておけば大丈夫なはずだが...
//もしなにかhandleMessageの処理を待ちたいとかあるならダメだけど
func (e *EdgeConnectionManager) messageHandler() {
	for {
		select {
		case conn := <-e.StreamCh():
			go e.handleMessage(conn)
		case <-e.shutdownCh:
			return
		}
	}
}

func (e *EdgeConnectionManager) GetHostFromConn(conn net.Conn) string {
	addr := conn.LocalAddr().String()
	temp := strings.SplitN(addr, ":", 2)
	return temp[0]
}

func (e *EdgeConnectionManager) handleMessage(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	conn.SetDeadline(time.Now().Add(e.TCPtimeout))
	n, err := conn.Read(buf)
	if err != nil {
		e.logger.Println(err)
		return
	}

	port, payload, msgType, err := e.messageManager.Parse(buf[:n])
	if err != nil {
		e.logger.Println(err)
		return
	}

	host := e.GetHostFromConn(conn)
	addr := getAddress(host, port)

	e.logger.Printf("messageType: %d,addr: %s,payload: %v\n", msgType, addr, payload)

	switch msgType {

	case MSG_PING:
		return
	case MSG_CORE_LIST:
		e.handleCoreList(payload)
		return
	default:
		e.logger.Println(ErrorUnknownMessage)
		return
	}

}

func (e *EdgeConnectionManager) handleCoreList(payload []byte) {
	e.logger.Println("Refresh Core Node List")
	mp, err := e.messageManager.GetPayload(payload)
	if err != nil {
		e.logger.Println(err)
		return
	}

	e.logger.Printf("latest Core Node List is: %v\n", mp.CoreNodeList)
	e.coreNodeList.Overwrite(mp.CoreNodeList)
}

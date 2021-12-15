package simplegocoin

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ConnectionManager struct {
	host             string
	port             string
	coreNodeList     *CoreNodeList
	edgeNodeList     *EdgeNodeList
	messageManager   *MessageManager
	shutdownCh       chan struct{}
	stopCh           chan struct{}
	streamCh         chan net.Conn
	TCPtimeout       time.Duration
	PINGinterval     time.Duration
	logger           *log.Logger
	listener         net.Listener
	tickers          []*time.Ticker
	handleBlockChain func(string, MessageType, Payload)
}

func NewConnectionManager(host, port string, TCPtimeout, PINGinterval time.Duration, logger *log.Logger) *ConnectionManager {
	mm := NewMessageManeger()
	cm := &ConnectionManager{
		host:           host,
		port:           port,
		coreNodeList:   NewCoreNodeList(logger),
		edgeNodeList:   NewEdgeNodeList(logger),
		messageManager: mm,
		shutdownCh:     make(chan struct{}),
		stopCh:         make(chan struct{}),
		streamCh:       make(chan net.Conn),
		logger:         logger,
		TCPtimeout:     TCPtimeout,
		PINGinterval:   PINGinterval,
	}

	cm.AddPeer(getAddress(host, port))

	return cm
}

//4んでいるノードをsliceに追加
func (c *ConnectionManager) Ping(addr string, deadNodes *[]string) {
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		c.logger.Println(err)
		*deadNodes = append(*deadNodes, addr)
		return
	}

	defer conn.Close()

	msg, err := c.messageManager.CreateWithoutPayload(c.port, MSG_PING)
	if err != nil {
		c.logger.Println(err)
		return
	}

	conn.Write(msg)
	//将来的にpingの結果が返ってきたらtrueとする

	return
}

//もしかしたらheartBeat自体にtimeoutやロックをかけて複数のheartBeatが同時に発火しないようにした方がいいかもしれない
func (c *ConnectionManager) HeartBeat() {
	c.logger.Println("HeartBeat was Called")

	deadNodes := []string{}
	//coreNodeList自体のLockもいるはず
	var wg sync.WaitGroup

	nodes := c.coreNodeList.CurrentNodes()

	wg.Add(len(nodes))

	//wgとchで制御
	for node, _ := range nodes {

		curNode := node
		if getAddress(c.host, c.port) == curNode {
			wg.Done()
			continue
		}
		go func() {
			c.Ping(curNode, &deadNodes)
			wg.Done()
		}()
	}

	wg.Wait()

	if len(deadNodes) == 0 {
		return
	}

	//4んでいるノードをリストから除外し、すべてのピアに通知
	for _, node := range deadNodes {
		c.coreNodeList.Remove(node)
	}

	payload := c.messageManager.CreateCoreListPayload(c.coreNodeList.CurrentNodes())
	bytes, err := payload.Marshal()
	if err != nil {
		c.logger.Println(err)
		return
	}

	msg, err := c.messageManager.Create(bytes, c.port, MSG_CORE_LIST)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.SendToAllPeer(msg)

}

func (c *ConnectionManager) Serve() {

	l, err := net.Listen(connectionProtocol, getAddress(c.host, c.port))
	if err != nil {
		log.Fatal(err)
		return
	}

	c.listener = l

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
			return

		}

		c.streamCh <- conn
	}
}

//待ち受けを開始する
//chを返して、Serveが終了したらそこで呼び出し元も終了させるGracefulShutDownはユーザー側の責務とする？
func (c *ConnectionManager) Run() {

	//メッセージ受信用のGoroutine
	go c.Serve()
	//メッセージ処理用のGoroutine
	go c.messageHandler()
	//heartBeat用のGoRoutine
	go c.schedule()

}

//ユーザーが指定したCoreノードへ接続
func (c *ConnectionManager) Join(host, port string) {
	addr := getAddress(host, port)
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		c.logger.Println(err)
		return
	}

	defer conn.Close()

	msg, err := c.messageManager.CreateWithoutPayload(c.port, MSG_ADD)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.Send(msg, addr)

}

//選択したノードに対しメッセージを送る
func (c *ConnectionManager) Send(msg []byte, addr string) {
	conn, err := net.Dial(connectionProtocol, addr)
	if err != nil {
		c.logger.Println(err)
		c.logger.Println("fail to connect peer")
		//本当はremoveの判定をもっと優しくしたりした方がいいはず(retryするとか、ゴシッププロトコルみたいにindirectProbeするとか)
		c.RemovePeer(addr)
		return
	}

	defer conn.Close()

	conn.Write(msg)
}

//coreNodeListに登録されているすべてのノードに対しメッセージを送る
func (c *ConnectionManager) SendToAllPeer(msg []byte) {
	c.logger.Println("Send Message to Allpeer...")
	for node, _ := range c.coreNodeList.CurrentNodes() {
		if getAddress(c.host, c.port) == node {
			//自分自身だったらcontinue
			continue
		}
		c.logger.Printf("Send Message To %s\n", node)
		c.Send(msg, node)
	}
}

func (c *ConnectionManager) Leave() {

}

//wgまでやってグレイスフルシャットダウンはいらない気がする?.memberlistとか普通の分散サーバーで一つのServerStructが複数のlistenerを立てるとかならいるのかな、でも今回p2pは1Serverにつき一人の想定なのでいらないかも
//shutDown,shutDownChはサーバー自体をシャットダウン、
//deschedule,stopChはheartbeatをstop
func (c *ConnectionManager) ShutDown() {
	close(c.shutdownCh)
	c.listener.Close() //これによりAcceptが新規受付しなくなる

	c.deschedule()
}

func (c *ConnectionManager) deschedule() {
	close(c.stopCh)
	//tickerをstop
	for _, t := range c.tickers {
		t.Stop()
	}
	c.tickers = nil
}

func (c *ConnectionManager) triggerFunc(duration time.Duration, C <-chan time.Time, stop <-chan struct{}, f func()) {

	for {
		select {
		case <-C:
			f()
		case <-stop:
			return
		}
	}

}

func (c *ConnectionManager) schedule() {
	// Create a new probeTicker
	if c.PINGinterval > 0 {
		t := time.NewTicker(c.PINGinterval)
		go c.triggerFunc(c.PINGinterval, t.C, c.stopCh, c.HeartBeat)
		c.tickers = append(c.tickers, t)
	}
}

func (c *ConnectionManager) StreamCh() <-chan net.Conn {
	return c.streamCh
}

//connectionManegerをmemberListのTransport部分とした方が良さそう？
//shutdownの時transport,つまりtcpのserveを閉じる(wgを使用)、そのあとはコネクション関係ないのでshutDownChを使って閉じるだけで良さそう
//この中のhandleMessageより先にServerが終わっても、shutdownChをcloseしておけば大丈夫なはずだが...
//もしなにかhandleMessageの処理を待ちたいとかあるならダメだけど
func (c *ConnectionManager) messageHandler() {
	for {
		select {
		case conn := <-c.StreamCh():
			go c.handleMessage(conn)
		case <-c.shutdownCh:
			return
		}
	}
}

func (c *ConnectionManager) GetHostFromConn(conn net.Conn) string {
	addr := conn.LocalAddr().String()
	temp := strings.SplitN(addr, ":", 2)
	return temp[0]
}

func (c *ConnectionManager) handleMessage(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	conn.SetDeadline(time.Now().Add(c.TCPtimeout * time.Millisecond))
	n, err := conn.Read(buf)
	if err != nil {
		c.logger.Println(err)
		return
	}

	port, payload, msgType, err := c.messageManager.Parse(buf[:n])
	if err != nil {
		c.logger.Println(err)
		return
	}

	host := c.GetHostFromConn(conn)
	addr := getAddress(host, port)

	c.logger.Printf("messageType: %d,addr: %s,payload: %v\n", msgType, addr, payload)

	switch msgType {
	case MSG_ADD:
		c.handleAdd(addr)
		return
	case MSG_REMOVE:
		c.handleRemove(addr)
		return
	case MSG_PING:
		return
	case MSG_REQUEST_CORE_LIST:
		c.handleRequestCoreList(addr)
		return
	case MSG_CORE_LIST:
		c.handleCoreList(payload)
		return
	case MSG_ADD_AS_EDGE:
		c.handleAddEdge(addr)
		return
	case MSG_REMOVE_EDGE:
		c.handleRemoveEdge(addr)
		return
	//ここからブロックチェーン絡み、serverに戻して処理
	case MSG_NEW_TRANSACTION, MSG_NEW_BLOCK, RSP_FULL_CHAIN, MSG_ENHANCED:
		payload, err := c.messageManager.GetBlockChainPayload(payload, msgType)
		if err != nil {
			c.logger.Println(err)
			return
		}
		c.handleBlockChain(addr, msgType, payload)
		return
	default:
		c.logger.Println(ErrorUnknownMessage)
		return
	}

}

func (c *ConnectionManager) SetHandleBlockChainFn(fn func(string, MessageType, Payload)) {
	c.handleBlockChain = fn
}

func (c *ConnectionManager) handleAddEdge(addr string) {
	c.edgeNodeList.Add(addr)
	//登録したedgeに対してcoreNodeListを返す(Edge->CoreのPINGの際に繋がらなかったら別のCoreに繋ぎ変えるために必要)
	payload := c.messageManager.CreateCoreListPayload(c.coreNodeList.CurrentNodes())
	bytes, err := payload.Marshal()
	if err != nil {
		c.logger.Println(err)
		return
	}

	msg, err := c.messageManager.Create(bytes, c.port, MSG_CORE_LIST)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.Send(msg, addr)
}

func (c *ConnectionManager) handleRemoveEdge(addr string) {
	c.edgeNodeList.Remove(addr)
}

func (c *ConnectionManager) handleRemove(addr string) {
	c.logger.Printf("Remove Request Received from %s\n", addr)
	c.RemovePeer(addr)
	payload := c.messageManager.CreateCoreListPayload(c.coreNodeList.CurrentNodes())
	bytes, err := payload.Marshal()
	if err != nil {
		c.logger.Println(err)
		return
	}

	msg, err := c.messageManager.Create(bytes, c.port, MSG_CORE_LIST)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.SendToAllPeer(msg)

}

func (c *ConnectionManager) handleRequestCoreList(addr string) {
	c.logger.Println("Core Node List was Requested")
	payload := c.messageManager.CreateCoreListPayload(c.coreNodeList.CurrentNodes())
	bytes, err := payload.Marshal()
	if err != nil {
		c.logger.Println(err)
		return
	}

	msg, err := c.messageManager.Create(bytes, c.port, MSG_CORE_LIST)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.Send(msg, addr)
}

func (c *ConnectionManager) handleCoreList(payload []byte) {
	c.logger.Println("Refresh Core Node List")
	ip, err := c.messageManager.GetCoreListPayload(payload)
	if err != nil {
		c.logger.Println(err)
		return
	}

	coreListPayload := ip.(*CoreListPayload)

	c.logger.Printf("latest Core Node List is: %v\n", coreListPayload.List)
	c.coreNodeList.Overwrite(coreListPayload.List)
}

func (c *ConnectionManager) handleAdd(addr string) {
	c.logger.Println("Add Request Received")
	c.AddPeer(addr)
	//neighborに現在のcore_node_listを一斉送信
	payload := c.messageManager.CreateCoreListPayload(c.coreNodeList.CurrentNodes())
	bytes, err := payload.Marshal()
	if err != nil {
		c.logger.Println(err)
		return
	}

	msg, err := c.messageManager.Create(bytes, c.port, MSG_CORE_LIST)
	if err != nil {
		c.logger.Println(err)
		return
	}
	c.SendToAllPeer(msg)

}

func getAddress(host, port string) string {
	return host + ":" + port
}

func (c *ConnectionManager) AddPeer(address string) {
	c.coreNodeList.Add(address)
}

func (c *ConnectionManager) RemovePeer(address string) {
	c.coreNodeList.Remove(address)
}

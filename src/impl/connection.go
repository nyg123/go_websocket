package impl

import (
	"errors"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"sync"
)

type Connection struct {
	wsConn    *websocket.Conn
	Id        string
	Room      string
	inChan    chan []byte
	outChan   chan []byte
	closeChan chan byte
	mutex     sync.Mutex
	isClose   bool
	Hub       *Hub
}

func IniConnection(WsConn *websocket.Conn, hub *Hub) (conn *Connection, err error) {
	conn = &Connection{
		wsConn:    WsConn,
		inChan:    make(chan []byte, 1000),
		outChan:   make(chan []byte, 1000),
		closeChan: make(chan byte, 1),
		Hub:       hub,
	}
	go conn.readLoop()
	go conn.writeLoop()
	return
}

func (conn *Connection) ReadMessage() (data []byte, err error) {
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("connection is close")
	}
	return
}

func (conn *Connection) WriteMessage(data []byte) (err error) {
	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("connection is close")
	}
	return
}

func (conn *Connection) Close() {
	glog.Infoln("关闭链接，id:", conn.Id, ",room:", conn.Room)
	_ = conn.wsConn.Close()
	conn.mutex.Lock()
	if !conn.isClose {
		if len(conn.Id) >= 1 {
			glog.Infoln("解绑id，id:", conn.Id, ",room:", conn.Room)
			conn.Hub.unregister <- conn
		}
		if len(conn.Room) >= 1 {
			glog.Infoln("解绑room，id:", conn.Id, ",room:", conn.Room)
			conn.Hub.LeaveRoom <- conn
		}
		close(conn.closeChan)
		conn.isClose = true
	}
	conn.mutex.Unlock()
}

//内部实现
func (conn *Connection) readLoop() {
	var (
		data []byte
		err  error
	)
	for {
		if _, data, err = conn.wsConn.ReadMessage(); err != nil {
			goto ERR
		}
		select {
		case conn.inChan <- data:
		case <-conn.closeChan:
			goto ERR
		}
	}
ERR:
	glog.Infoln("error")
	conn.Close()
}

func (conn *Connection) writeLoop() {
	var (
		data []byte
		err  error
	)
	for {
		select {
		case data = <-conn.outChan:
		case <-conn.closeChan:
			goto ERR
		}
		if err = conn.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			goto ERR
		}
	}
ERR:
	glog.Infoln("error")
	conn.Close()
}

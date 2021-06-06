package tool

import (
	"encoding/json"
	"github.com/golang/glog"
)

// Msg data format pushed to client
type Msg struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// MsgTmp ServerMsg
type MsgTmp struct {
	Id   string
	Room string
	Msg  Msg
}

type Hub struct {
	Clients map[string]*Connection

	Rooms map[string]map[*Connection]bool

	BroadcastId chan *MsgTmp

	BroadcastRoom chan *MsgTmp

	Register chan *Connection

	Unregister chan *Connection

	//加入房间
	AddRoom chan *Connection

	//离开房间
	LeaveRoom chan *Connection
}

func NewHub() *Hub {
	return &Hub{
		BroadcastId:   make(chan *MsgTmp, 1000),
		BroadcastRoom: make(chan *MsgTmp, 1000),
		Register:      make(chan *Connection, 1000),
		Unregister:    make(chan *Connection, 1000),
		AddRoom:       make(chan *Connection, 1000),
		LeaveRoom:     make(chan *Connection, 1000),
		Clients:       make(map[string]*Connection),
		Rooms:         make(map[string]map[*Connection]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.Register:
			if old, ok := h.Clients[conn.Id]; ok {
				old.Close()
			}
			h.Clients[conn.Id] = conn
		case conn := <-h.AddRoom:
			if _, ok := h.Rooms[conn.Room]; !ok {
				h.Rooms[conn.Room] = map[*Connection]bool{}
			}
			h.Rooms[conn.Room][conn] = true
		case conn := <-h.Unregister:
			if old, ok := h.Clients[conn.Id]; ok {
				if old.Uuid == conn.Uuid {
					delete(h.Clients, conn.Id)
				}
			}
		case conn := <-h.LeaveRoom:
			if _, ok := h.Rooms[conn.Room][conn]; ok {
				delete(h.Rooms[conn.Room], conn)
			}
		case message := <-h.BroadcastId:
			conn, ok := h.Clients[message.Id]
			marshal, _ := json.Marshal(message.Msg)
			if ok {
				err := conn.WriteMessage(marshal)
				if err != nil {
					glog.Infoln("send message error", err)
					delete(h.Clients, conn.Id)
				} else {
					glog.Infoln("send message，id:", message.Id, "，data:", string(marshal))
				}
			}
		case message := <-h.BroadcastRoom:
			rooms, ok := h.Rooms[message.Room]
			marshal, _ := json.Marshal(message.Msg)
			if ok {
				go func() {
					for conn, _ := range rooms {
						err := conn.WriteMessage(marshal)
						if err != nil {
							glog.Errorf("Failed to send message，message:%+v,error:%+v", message, err)
						}
					}
					glog.Infof("发送消息成功，message:%+v,连接数量：%d", message, len(rooms))
				}()
			}
		}
	}
}

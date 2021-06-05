package impl

import (
	"encoding/json"
	"github.com/golang/glog"
)

type Msg struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type Hub struct {
	clients map[string]*Connection

	rooms map[string]map[*Connection]bool

	Broadcast chan *MsgId

	RoomBroadcast chan *MsgRoom

	Register chan *Connection

	unregister chan *Connection

	//加入房间
	AddRoom chan *Connection

	//离开房间
	LeaveRoom chan *Connection
}

type MsgId struct {
	Id  string
	Msg Msg
}

type MsgRoom struct {
	Room string
	Msg  Msg
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:     make(chan *MsgId, 1000),
		RoomBroadcast: make(chan *MsgRoom, 1000),
		Register:      make(chan *Connection, 1000),
		unregister:    make(chan *Connection, 1000),
		AddRoom:       make(chan *Connection, 1000),
		LeaveRoom:     make(chan *Connection, 1000),
		clients:       make(map[string]*Connection),
		rooms:         make(map[string]map[*Connection]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.Register:
			c, ok := h.clients[conn.Id]
			if ok {
				c.Close()
			}
			h.clients[conn.Id] = conn
		case conn := <-h.AddRoom:
			if _, ok := h.rooms[conn.Room]; !ok {
				h.rooms[conn.Room] = map[*Connection]bool{}
			}
			h.rooms[conn.Room][conn] = true
		case conn := <-h.unregister:
			if _, ok := h.clients[conn.Id]; ok {
				delete(h.clients, conn.Id)
			}
		case conn := <-h.LeaveRoom:
			if _, ok := h.rooms[conn.Room][conn]; ok {
				delete(h.rooms[conn.Room], conn)
			}
		case message := <-h.Broadcast:
			conn, ok := h.clients[message.Id]
			marshal, _ := json.Marshal(message.Msg)
			if ok {
				go func() {
					select {
					case conn.outChan <- marshal:
					default:
						conn.Close()
					}
				}()
				glog.Infoln("发送消息，id:", message.Id, "，数据:", string(marshal))
			}
		case message := <-h.RoomBroadcast:
			rooms, ok := h.rooms[message.Room]
			marshal, _ := json.Marshal(message.Msg)
			if ok {
				go func() {
					for conn, _ := range rooms {
						if conn.isClose {
							conn.Close()
							continue
						}
						select {
						case conn.outChan <- marshal:
						default:
							conn.Close()
						}
					}
					glog.Infoln("发送消息，room:", message.Room, "，数据:", string(marshal), "，链接数:", len(rooms))
				}()
			}
		}
	}
}

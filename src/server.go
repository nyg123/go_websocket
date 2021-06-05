package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	impl2 "go_websocket/src/impl"
	"net/http"
)

const (
	EventHeartBeat = "heartbeat"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	hub = impl2.NewHub()
)

var port = flag.String("port", "80", "端口号")
var certFile = flag.String("certFile", "", "cert文件路径")
var keyFile = flag.String("keyFile", "", "key文件路径")

func main() {
	//初始化命令行参数
	flag.Parse()
	defer glog.Flush()
	glog.Infoln("开始启动")
	go hub.Run()
	glog.Infoln("启动处理消息协程完成")
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/sendById", sendId)
	http.HandleFunc("/sendByRoom", sendRoom)
	glog.Infoln("绑定路由完成")
	url := fmt.Sprintf("0.0.0.0:%s", *port)
	glog.Infoln("启动服务：", url)
	if err := http.ListenAndServeTLS(url, *certFile, *keyFile, nil); err != nil {
		glog.Error("启动失败", err)
	}

}

func send(conn *impl2.Connection, msg *impl2.Msg) {
	marshal, err := json.Marshal(msg)
	if err != nil {
		glog.Errorln("发送数据错误:", err)
		return
	}
	if err := conn.WriteMessage(marshal); err != nil {
		glog.Errorln("发送数据错误:", err)
		return
	}
}

// websocket 路由
func wsHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infoln("开始websocket握手")
	var (
		WsConn *websocket.Conn
		conn   *impl2.Connection
		err    error
		data   []byte
	)
	id := r.FormValue("id")
	room := r.FormValue("room")
	if WsConn, err = upgrader.Upgrade(w, r, nil); err != nil {
		return
	}
	if conn, err = impl2.IniConnection(WsConn, hub); err != nil {
		goto ERR
	}

	if id != "" {
		conn.Id = id
		conn.Hub.Register <- conn
		glog.Infoln("绑定用户id成功，id:", id)
	}

	if room != "" {
		conn.Room = room
		conn.Hub.AddRoom <- conn
		glog.Infoln("绑定房间成功，room:", room)
	}

	for {
		if data, err = conn.ReadMessage(); err != nil {
			goto ERR
		}
		glog.Infoln("接收到消息:", string(data))
		inData := impl2.Msg{}
		err := json.Unmarshal(data, &inData)
		if err != nil {
			glog.Errorln("解析数据错误:", err)
		}
		if inData.Event == EventHeartBeat {
			send(conn, &inData)
		}
	}
ERR:
	conn.Close()
}

// 单推路由
func sendId(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.FormValue("id")
	data := r.FormValue("data")
	event := r.FormValue("event")
	if id == "" {
		_, err := w.Write([]byte(`{"status":"error","msg":"id 不能为空"}`))
		if err != nil {
			glog.Error(err)
		}
		return
	}
	msgId := impl2.MsgId{
		Id: id, Msg: impl2.Msg{
			Event: event,
			Data:  data,
		},
	}
	hub.Broadcast <- &msgId
	_, err := w.Write([]byte(`{"status":"success"}`))
	if err != nil {
		glog.Error(err)
	}
}

// 群推路由
func sendRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	room := r.FormValue("room")
	data := r.FormValue("data")
	event := r.FormValue("event")
	if room == "" {
		_, err := w.Write([]byte(`{"status":"error","msg":"room 不能为空"}`))
		if err != nil {
			glog.Error(err)
		}
		return
	}
	msgRoom := impl2.MsgRoom{
		Room: room, Msg: impl2.Msg{
			Event: event,
			Data:  data,
		},
	}
	hub.RoomBroadcast <- &msgRoom
	_, err := w.Write([]byte(`{"status":"success"}`))
	if err != nil {
		glog.Error(err)
	}
}

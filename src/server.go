package main

import (
	"encoding/json"
	"flag"
	"fmt"
	idworker "github.com/gitstliu/go-id-worker"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/configor"
	"main/tool"
	"net/http"
	"runtime"
	"strconv"
)

const (
	EventHeartBeat = "heartbeat"
)

var configPath = flag.String("c", "conf.json", "配置文件")

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	hub        = tool.NewHub()
	CurrWorker = &idworker.IdWorker{}
)

var Config = struct {
	Port     string `default:"80"`
	TLS      bool   `default:"false"`
	CertFile string
	KeyFile  string
}{}

func init() {
	//初始化命令行参数
	flag.Parse()
	err := configor.Load(&Config, *configPath)
	if err != nil {
		panic(err)
	}
	err = CurrWorker.InitIdWorker(1000, 1)
	if err != nil {
		panic(err)
	}
}

func main() {
	//初始化命令行参数
	flag.Parse()
	defer glog.Flush()
	glog.Infoln("Start ...")
	go hub.Run()
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/sendById", sendById)
	http.HandleFunc("/sendByRoom", sendRoom)
	http.HandleFunc("/monitor", monitor)
	url := fmt.Sprintf(":%s", Config.Port)
	if Config.TLS {
		glog.Infof("Start TLS service success, visit https://127.0.0.1:%s", Config.Port)
		if err := http.ListenAndServeTLS(url, Config.CertFile, Config.KeyFile, nil); err != nil {
			glog.Error("Failed to start", err)
		}
	} else {
		if err := http.ListenAndServe(url, nil); err != nil {
			glog.Error("Failed to start", err)
		}
	}

}

// Send data to client
func send(conn *tool.Connection, msg *tool.Msg) {
	marshal, err := json.Marshal(msg)
	if err != nil {
		glog.Errorln("Sending data error:", err)
		return
	}
	if err := conn.WriteMessage(marshal); err != nil {
		glog.Errorln("Sending data error:", err)
		return
	}
}

// websocket 路由
func wsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		WsConn *websocket.Conn
		conn   *tool.Connection
		err    error
		data   []byte
	)
	id := r.FormValue("id")
	room := r.FormValue("room")
	if WsConn, err = upgrader.Upgrade(w, r, nil); err != nil {
		return
	}
	uuid, _ := CurrWorker.NextId()
	if conn, err = tool.IniConnection(WsConn, uuid, hub); err != nil {
		goto ERR
	}

	if id != "" {
		conn.Id = id
		hub.Register <- conn
		glog.Infoln("Bind user ID succeeded，id:", id)
	}

	if room != "" {
		conn.Room = room
		hub.AddRoom <- conn
		glog.Infoln("Bind room succeeded，room:", room)
	}

	for {
		if data, err = conn.ReadMessage(); err != nil {
			goto ERR
		}
		glog.Infoln("receive message:", string(data))
		inData := tool.Msg{}
		err := json.Unmarshal(data, &inData)
		if err != nil {
			glog.Errorln("Parsing data error:", err)
		}
		// When receiving the heartbeat packet from the client, reply to the heartbeat packet
		if inData.Event == EventHeartBeat {
			send(conn, &inData)
		}
	}
ERR:
	glog.Infoln("close websocket")
	conn.Close()
}

//Push messages to a single user
func sendById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.FormValue("id")
	data := r.FormValue("data")
	event := r.FormValue("event")
	if id == "" {
		_, err := w.Write([]byte(`{"status":"error","msg":"id Cannot be empty"}`))
		if err != nil {
			glog.Error(err)
		}
		return
	}
	msgTmp := tool.MsgTmp{
		Id: id, Msg: tool.Msg{
			Event: event,
			Data:  data,
		},
	}
	hub.BroadcastId <- &msgTmp
	_, err := w.Write([]byte(`{"status":"success"}`))
	if err != nil {
		glog.Error(err)
	}
}

// Push message to room
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
	msgTmp := tool.MsgTmp{
		Room: room, Msg: tool.Msg{
			Event: event,
			Data:  data,
		},
	}
	hub.BroadcastRoom <- &msgTmp
	_, err := w.Write([]byte(`{"status":"success"}`))
	if err != nil {
		glog.Error(err)
	}
}

// monitor
func monitor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var data = make(map[string]interface{})
	data["connect_num"] = strconv.Itoa(len(hub.Clients))
	data["BroadcastId_len"] = strconv.Itoa(len(hub.BroadcastId))
	data["BroadcastRoom_len"] = strconv.Itoa(len(hub.BroadcastRoom))
	data["Register_len"] = strconv.Itoa(len(hub.Register))
	data["unregister_buffer"] = strconv.Itoa(len(hub.Unregister))
	data["AddRoom_buffer"] = strconv.Itoa(len(hub.AddRoom))
	data["LeaveRoom_buffer"] = strconv.Itoa(len(hub.LeaveRoom))
	data["mem"] = fmt.Sprintf("%dM", m.Sys/1024/1024)
	var room = make(map[string]interface{})
	for index, r := range hub.Rooms {
		var ids []string
		for id := range r {
			ids = append(ids, id.Id)
		}
		room[index] = ids
	}
	data["room"] = room
	marshal, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, err = w.Write(marshal)
	if err != nil {
		glog.Error(err)
	}
}

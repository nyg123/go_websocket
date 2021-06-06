package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

var addr = flag.String("addr", "127.0.0.1:80", "http service address")
var num = flag.Int("num", 1000, "http service number")

var interrupt = make(chan os.Signal, 1)
var lock sync.Mutex

var total int

var max = 1
var w = sync.WaitGroup{}

type Msg struct {
	Event string `json:"event"`
	Data  int    `json:"data"`
}

func main() {
	flag.Parse()
	signal.Notify(interrupt, os.Interrupt)
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	glog.Infof("connecting to %s", u.String())

	w.Add(*num)
	for i := 1; i <= *num; i++ {
		u.RawQuery = fmt.Sprintf("id=%s&room=a", strconv.Itoa(i))
		lock.Lock()
		go create(u.String(), i)
	}
	w.Wait()
}

func create(urlString string, i int) {
	c, _, err := websocket.DefaultDialer.Dial(urlString, nil)
	if err != nil {
		glog.Errorln("dial:", err)
		time.Sleep(10 * time.Second)
		lock.Unlock()
		return
	}
	lock.Unlock()
	defer func(c *websocket.Conn) {
		w.Done()
		err := c.Close()
		if err != nil {

		}
	}(c)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				glog.Infoln("read:", err)
				wg.Done()
				return
			}
			total++
			if i > max {
				max = i
			}
			inData := make(map[string]string)
			glog.Infof("接收到消息%s,id:%d,当前链接数:%d,消息总数：%d", string(message), i, max, total)
			err = json.Unmarshal(message, &inData)
			if err != nil {
				glog.Infoln(err)
			}
			if inData["data"] != strconv.Itoa(i) {
				glog.Infoln(inData, i)
				wg.Done()
				return
			}

		}
	}()
	var msg = fmt.Sprintf(` {"event": "heartbeat", "data":"%d"}`, i)
	go func() {
		for {
			err := c.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				glog.Infoln("接收到消息", err)
				wg.Done()
				return
			}
			time.Sleep(time.Second)
		}
	}()
	wg.Wait()
	return
}

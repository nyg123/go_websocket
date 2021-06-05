package main

import (
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

var addr = flag.String("addr", "127.0.0.1:7777", "http service address")
var num = flag.Int("num", 1000, "http service number")

var interrupt = make(chan os.Signal, 1)
var lock sync.Mutex

var msg = `{
    "event": "heartbeat",
    "data": {
        "time": 2021
    }
}`
var total int

var max = 1

func main() {
	flag.Parse()
	signal.Notify(interrupt, os.Interrupt)
	u := url.URL{Scheme: "wss", Host: *addr, Path: "/ws"}
	glog.Infof("connecting to %s", u.String())
	for i := 1; i <= *num; i++ {
		u.RawQuery = fmt.Sprintf("id=%s&room=a", strconv.Itoa(i))
		lock.Lock()
		go create(u.String(), i)
	}
	time.Sleep(time.Hour)
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
		lock.Unlock()
		err := c.Close()
		if err != nil {

		}
	}(c)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				glog.Infoln("read:", err)
				return
			}
			total++
			if i > max {
				max = i
			}
			glog.Infof("接收到消息%s,id:%d,当前链接数:%d,消息总数：%d", string(message), i, max, total)
		}
	}()
	/*go func() {
		for {
			err := c.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				glog.Infoln("接收到消息", err)
				return
			}
			time.Sleep(60 * time.Second)
		}
	}()*/

	for {
		select {
		case <-done:
			return
		}
	}
}

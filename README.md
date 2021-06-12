## 介绍 
> 这是一个用go写的websocket服务端,并对外提供http接口。

> 客户端连接websocket 时，可以绑定用户id和房间room。

> 其他服务可以通过调用http接口，给某个id或room推送消息。

> 支持ws和wss协议，微信小程序必须使用wss协议

## 文件目录

```
cmd  //可以直接运行的编译后软件
--conf.json //配置文件
--server   //生成的linux环境下的编译文件
--server.exe //windows环境下的
src //服务端源码
test //客户端模拟测试工具
```

## 配置说明 `conf.json`

```
{
    "Port" : "80", //开放的端口号
    "TLS" : false, //是否开启tls，不开启是ws协议，开启是wss协议
    "CertFile" : "", //证书(PEM格式)文件路径，开启wss时一定要有证书
    "KeyFile" : "" //密钥（KEY）文件，开启wss时一定要有证书
}
```

## 启动以windows为例

```
cd cmd
./server.exe -c conf.json
 
 // 如果需要把日志输出到文件，可以指定-log_dir ,文件夹必须存在。如
 ./server -c conf.json -log_dir ./logs/ 
```

## 客户端连接测试
  ![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606122143.png)


## 客户端发送心跳包 
  
#### 心跳包格式
```json
{"event":"heartbeat","data":"这里是我自己定义的数据"}
```
#### 服务端收到心跳包时，会原样返回回来
![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606122733.png)

#### 通过用户id进行推送
客户端建立连接时，传送id，就会进行绑定,比如绑定id 123
  ![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606123150.png)
  
#### 服务端通过调用/sendById接口进行推送
![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606123431.png)

客户端收到消息
![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606123714.png)

#### 通过房间room进行推送，群推
和单推是类似的，客户端建立连接时绑定room
  > 127.0.0.1:80/ws?id=123&room=abc

#### 推送时调用
> http://127.0.0.1:80/sendByRoom 传递参数为`room`、`event`、`data`

![avatar](https://afw656.oss-cn-beijing.aliyuncs.com/myfile/QQ%E6%88%AA%E5%9B%BE20210606124336.png)

package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户的列表
	mapLock   sync.RWMutex
	OnlineMap map[string]*User

	//消息广播的channel
	Message chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息channel的goruntine，一旦有消息就发送给user
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message

		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

// BroadCast 广播消息的方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

func (this *Server) Handler(conn net.Conn) {
	//当前链接的业务
	user := NewUser(conn, this)

	user.Online()
	//接受客户端发送的消息

	//监听用户是否活跃的channel
	isLive := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户的消息(去除"\n")
			msg := string(buf[:n-1])

			//用户针对msg进行消息处理
			user.DoMessage(msg)

			//用户的任意消息代表用户是活跃的
			isLive <- true
		}
	}()

	//当前handler阻塞
	for {
		select {
		case <-isLive:
			//当前用户活跃，应该重置定时器
			//不做任何事情，为了激活select，更新下面的定时器
		case <-time.After(time.Second * 60 * 5):
			//已经超时
			//将当前客户端强制关闭

			user.SendMsg("超时请重新连接")

			//销毁资源
			close(user.C)

			//关闭连接
			conn.Close()
			//退出
			return
		}

	}
}

//启动服务器的接口

func (this *Server) Start() {

	//socket listen
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err")
	}
	//close listen socket
	defer listen.Close()
	//启动监听Message的goruntine
	go this.ListenMessage()

	for {
		//accept
		accept, err := listen.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		//do handler
		go this.Handler(accept)
	}

}
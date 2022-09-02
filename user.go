package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	Server *Server
}

// NewUser 创建一个用户
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		Server: server,
	}
	//启动监听当前user
	go user.ListenMessage()
	return user
}

// 用户上线
func (this *User) Online() {

	//用户上线了,加入map中
	this.Server.mapLock.Lock()
	this.Server.OnlineMap[this.Name] = this
	this.Server.mapLock.Unlock()
	//广播当前用户上线消息
	this.Server.BroadCast(this, "已上线")
}

// 用户下线
func (this *User) Offline() {

	this.Server.mapLock.Lock()
	delete(this.Server.OnlineMap, this.Name)
	this.Server.mapLock.Unlock()
	//广播当前用户上线消息
	this.Server.BroadCast(this, "下线")
}

// 给当前用户对应的客户端发送消息
func (this User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

// 用户处理消息的业务
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户有哪些
		this.Server.mapLock.Lock()
		for _, user := range this.Server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMsg(onlineMsg)
		}
		this.Server.mapLock.Unlock()

	} else if len(msg) > 7 && msg[:7] == "rename|" {
		//消息格式: rename|张三
		newName := strings.Split(msg, "|")[1]

		//判断name是否存在
		_, ok := this.Server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名已被使用\n")
		} else {
			this.Server.mapLock.Lock()
			delete(this.Server.OnlineMap, this.Name)
			this.Server.OnlineMap[newName] = this
			this.Server.mapLock.Unlock()

			this.Name = newName
			this.SendMsg("您已更新用户名为" + newName + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式: to|张三|消息格式

		//1 获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMsg("消息格式不正确，请使用 to|张三|消息 格式\n")
			return
		}
		//2 根据用户名得到User对象
		remoteUser, ok := this.Server.OnlineMap[remoteName]

		if !ok {
			this.SendMsg("该用户不存在\n")
			return
		}
		//3 获取消息内容，将消息发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMsg("消息内容为空\n")
			return
		}
		remoteUser.SendMsg(this.Name + "对您说：" + content + "\n")
		this.SendMsg("发送成功!\n")
	} else {
		this.Server.BroadCast(this, msg)
	}
}

// ListenMessage 监听User chan 的方法，如果有消息就发给客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C
		//写给客户端
		this.conn.Write([]byte(msg + "\n"))
	}
}

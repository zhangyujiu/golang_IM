package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	go user.ListenMessage()
	return user
}

func (this *User) ListenMessage() {
	for {
		msg := <-this.C
		this.conn.Write([]byte(msg + "\n"))
	}
}

func (this *User) Online() {
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	//广播用户上线消息
	this.server.Broadcast(this, "online...")
}
func (this *User) Offline() {
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	//广播用户下线消息
	this.server.Broadcast(this, "offline...")
}

func (this *User) DealWithMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户有哪些
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "online...\n"
			this.SendMessage(onlineMsg)
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		//判断name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMessage("Username is taken")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()
			this.Name = newName
			this.SendMessage("Username modified successfully:" + this.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式 to|张三|消息
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMessage("Incorrect message format")
			return
		}
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.SendMessage("User does not exist")
			return
		}
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMessage("Message cannot be empty")
			return
		}
		remoteUser.SendMessage(this.Name + ":" + content)
	} else {
		this.server.Broadcast(this, msg)
	}

}
func (this *User) SendMessage(msg string) {
	this.conn.Write([]byte(msg))
}

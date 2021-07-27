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

	//在线用户
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播的channel
	Message chan string
}

func (this *Server) Handler(conn net.Conn) {
	//用户上线了
	user := NewUser(conn, this)
	user.Online()

	//监听用户是否活跃的channel
	isLive := make(chan bool)

	//接受客户端发送的消息
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

			//提取用户消息（去除'\n'）
			msg := string(buf[:n-1])
			user.DealWithMessage(msg)

			//用户的任意消息，代表当前用户是一个活跃的用户
			isLive <- true
		}
	}()

	for {
		timeChan := time.After(time.Second * 300)
		select {
		case <-isLive:
			//当前用户活跃，需要重置定时器
			//不做任何操作，为了激活select，更新下面的定时器
		case <-timeChan:
			//超时
			user.SendMessage("You got kicked off the line")
			close(user.C)
			conn.Close()
			delete(this.OnlineMap, user.Name)
			return
		}
	}
}

func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message

		this.mapLock.Lock()
		for _, v := range this.OnlineMap {
			v.C <- msg
		}
		this.mapLock.Unlock()
	}
}

func (this *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

//创建一个server
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

//启动服务器
func (this *Server) Start() {
	//开启socket
	listener, err := net.Listen("tcp", fmt.Sprint(this.Ip, ":", this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	//关闭socket
	defer listener.Close()

	go this.ListenMessage()

	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}
		go this.Handler(conn)
	}
}

package main

/*
 * @Description:
 * @Author: Liuhongq
 * @Date: 2022-03-31 04:01:09
 * @LastEditTime: 2022-04-09 05:32:46
 * @LastEditors: Liuhongq
 * @Reference:
 */

import (
	"fmt"
	"net"
	"time"
)

type Client struct {
	C    chan string
	Name string
	Addr string
}

//在线用户
var OnlineMap = make(map[string]Client)

//公共通知消息
var message = make(chan string)

func main() {
	listenner, err := net.Listen("tcp", "127.0.0.1:8010")

	if err != nil {
		fmt.Println("net.Listen err: ", err)
		return
	}

	defer listenner.Close()
	//转发消息
	go NotifyMsg()
	//持续监听阻塞
	for {
		conn, err := listenner.Accept()
		if err != nil {
			fmt.Println("listenner.Accept err: ", err)
			continue
		}
		//多并发登录
		go Login(conn)
	}
}

func Login(conn net.Conn) {
	defer conn.Close()
	//检测用户状态
	isData := make(chan bool)
	isExit := make(chan bool)

	addr := conn.RemoteAddr().String()

	client := Client{make(chan string), addr, addr}

	//将所有上线用户存入map
	OnlineMap[addr] = client

	message <- MakeMsg(client, "Login!!")

	//接收客户端信息
	go ReadMsg(client, conn, isData, isExit)

	go SendMsgToClient(client, conn)

	//超时处理
	for {
		select {
		case <-isData:
			break
		case <-isExit:
			delete(OnlineMap, addr)
			message <- MakeMsg(client, "Exit!")
			return
		case <-time.After(60 * time.Second):
			delete(OnlineMap, addr)
			message <- MakeMsg(client, "Time Out Exit!")
			return
		}
	}

}

func MakeMsg(client Client, text ...string) string {
	msg := fmt.Sprintf("[%s]:[%s]==>%v\n", client.Addr, client.Name, text)
	return msg
}

//接收客户端信息
func ReadMsg(client Client, conn net.Conn, isData chan bool, isExit chan bool) {
	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			//fmt.Println("conn.Read err: ", err)
			isExit <- true
			return
		}
		if n == 0 {
			fmt.Println("ReadMsg End!!!")
			return
		}
		msg := string(buf[:n-1])

		if len(msg) == 3 && msg == "who" {
			for _, v := range OnlineMap {
				client.C <- MakeMsg(v, "Online")
			}
		} else if len(msg) > 7 && msg[:7] == "rename " {
			oldName := client.Name
			newName := msg[7:]
			client.Name = newName
			OnlineMap[client.Addr] = client
			client.C <- MakeMsg(client, "Rename Succes!", oldName, "======>", newName)
		} else {
			message <- MakeMsg(client, msg)
		}
		isData <- true
	}

}

//给当前客户端发送消息
func SendMsgToClient(client Client, conn net.Conn) {
	for msg := range client.C {
		conn.Write([]byte(msg))
	}
}

//转发消息给所有在线用户
func NotifyMsg() {
	for {
		msg := <-message
		for _, client := range OnlineMap {
			client.C <- msg
		}
	}
}

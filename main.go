package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)


var upgrader = websocket.Upgrader{
	// 允许跨域（开发时使用）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientIP := r.RemoteAddr
	fmt.Println("Client connected","remote addr", clientIP)

	for {
		// 读取消息
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err, "close this socket")
			return
		}
		
		textMsg := string(msg)
		fmt.Printf("msgType = %d recv: %s\n",msgType, textMsg)
		// 回写消息（echo）
		if err := conn.WriteMessage(msgType, msg); err != nil {
			fmt.Println("Write error:", err)
			break
		}

		switch msgType {
		case websocket.TextMessage: //文本消息
			handleTextMsg(textMsg, conn)
		case websocket.BinaryMessage: //二进制消息
			handleBinaryMsg(textMsg, conn)
		default:
			fmt.Println("handle default")
		}
	}
}

func handleTextMsg(textMsg string, conn *websocket.Conn){
	fmt.Println("handle msg", textMsg)

	switch textMsg {
	case "play":
		AudioPlayPcm(conn)
	case "playopus":
		AudioPlayOpus(conn)
	}
}

func handleBinaryMsg(textMsg string, conn *websocket.Conn){
	fmt.Println("handle msg", textMsg)
	conn.WriteMessage(websocket.TextMessage,[]byte("play music"));
}

func main(){
	var port string = ":8080"
	fmt.Println("This is launcher server!")

	http.HandleFunc("/ws", wsHandler)
	fmt.Println("WebSocket server started: ws://localhost"+ port +"/ws")
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Println(err)
	}
}
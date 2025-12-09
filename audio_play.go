package main

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/go-mp3"
)

func AudioPlay(conn *websocket.Conn) {
	fmt.Println("Auido play...")

	conn.WriteMessage(websocket.TextMessage, []byte("music_play"))

	//decode mp3 file
	audioFile, err := os.Open("audio/san.mp3")
	if err != nil {
		fmt.Println(err)
		return
	}

	decoder, err := mp3.NewDecoder(audioFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("file %d sample %d\n", decoder.Length(), decoder.SampleRate())
	buf := make([]byte, 4*1024) //4k buffer
	sendBytes := 0
	for {
		n, err := decoder.Read(buf)

		if n > 0 {
			err = conn.WriteMessage(websocket.BinaryMessage, buf[0:n])
			sendBytes += n
			fmt.Println("send",sendBytes,"/",decoder.Length())
			// time.Sleep(200 * time.Millisecond)

			if err != nil {
				fmt.Println("WS write:", err)
				conn.WriteMessage(websocket.TextMessage, []byte("musicstop"))
				break
			}
		}
		
		if err != nil {
			fmt.Println("mp3 decode end:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("musicstop"))
			break
		}
	}
}
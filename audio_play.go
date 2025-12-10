package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/go-mp3"
	"gopkg.in/hraban/opus.v2"
)

func AudioPlayOpus(conn *websocket.Conn) {
	fmt.Println("Auido play decoder opus...")
	conn.WriteMessage(websocket.TextMessage, []byte("music_play_opus"))
    
	f, err := os.Open("audio/san.mp3")
	if err != nil {
        panic(err)
    }
    defer f.Close()

    // 解码为 PCM
    dec, err := mp3.NewDecoder(f)
    if err != nil {
		fmt.Println("mp3 decode error")
        return
    }

    samplerate := dec.SampleRate()
    channels := 2 // go-mp3 默认输出 stereo (通常 MP3 都是 stereo)

    fmt.Println("MP3 sample rate:", samplerate)
    // 创建 Opus Encoder
    enc, err := opus.NewEncoder(samplerate, channels, opus.AppAudio)
    if err != nil {
		fmt.Println("opus encode create error")
        return
    }

    // 设置目标码率（可调）
    enc.SetBitrate(64000)

    // 每帧 20ms
    frameSize := samplerate / 50 // 20ms = 1/50 秒

    // 输出文件
    out, err := os.Create("output.opus")
    if err != nil {
        panic(err)
    }
    defer out.Close()

    // 每次读取一帧 PCM 数据
    bufSize := frameSize * channels * 2 // 16-bit PCM -> 每样本 2 字节

    pcmBuf := make([]byte, bufSize)
    opusBuf := make([]byte, 4000) // opus 编码输出缓冲

    for {
        n, err := io.ReadFull(dec, pcmBuf)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            fmt.Println("File read done.")
            break
        }
        if err != nil {
			fmt.Println("Read file error!")
            break
        }
        if n < len(pcmBuf) {
            // 不足一帧则丢弃（通常只在结尾）
            break
        }

        // PCM16 → float32
        pcmF := pcm16ToFloat32(pcmBuf)

        // Opus 编码
        encoded, err := enc.EncodeFloat32(pcmF, opusBuf)
        if err != nil {
            fmt.Println("Opus encode error")
            break
        }

        // encoded 是有效的编码字节数
		//
        _, err = out.Write(opusBuf[:encoded])
        if err != nil {
            panic(err)
        }
    }
    fmt.Println("Output is written to output.opus")
}

// 将 16-bit PCM 转为 float32（Opus 编码器需要）
func pcm16ToFloat32(src []byte) []float32 {
    samples := make([]float32, len(src)/2)
    for i := 0; i < len(samples); i++ {
        v := int16(binary.LittleEndian.Uint16(src[i*2:]))
        samples[i] = float32(v) / 32768.0
    }
    return samples
}

func AudioPlayPcm(conn *websocket.Conn) {
	fmt.Println("Auido play pcm...")

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
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/go-mp3"
	"gopkg.in/hraban/opus.v2"
)

// func AudioPlayOpus(conn *websocket.Conn) {
// 	fmt.Println("Auido play decoder opus...")
// 	conn.WriteMessage(websocket.TextMessage, []byte("music_play_opus"))

// 	// f, err := os.Open("audio/san.mp3")
// 	f, err := os.Open("audio/guang.mp3")
// 	if err != nil {
//         fmt.Println("read file error", err.Error())
//         return
//     }

//     defer f.Close()

//     // 解码为 PCM
//     decoder, err := mp3.NewDecoder(f)
//     if err != nil {
// 		fmt.Println("mp3 decode error", err.Error())
//         return
//     }

//     srcRate := decoder.SampleRate() //原始采样率
//     channels := 2 // go-mp3 默认输出 stereo (通常 MP3 都是 stereo)
//     fmt.Println("MP3 sample rate:", srcRate, decoder.Length())

//     outRate := 48000 //opus推荐采样率

//     // 创建 Opus Encoder
//     enc, err := opus.NewEncoder(outRate, channels, opus.AppAudio)
//     if err != nil {
// 		fmt.Println("opus encode create error", err.Error())
//         return
//     }
//     // 设置目标码率（可调）
//     enc.SetBitrate(64000)

//     // 每帧 20ms
//     frameSize := outRate * 20 / 1000
//     samplesPerFrame := frameSize * channels

//     readBuf := make([]byte, 8 * 1024) // MP3 解码读取 buffer
//     pcmBuf := make([]int16, 0)    // 累积若干 PCM samples

//     ticker := time.NewTicker(20 * time.Millisecond)
// 	defer ticker.Stop()

// 	totalPcmBytes := decoder.Length()
// 	sendPcmBytes := 0
//     for{
//         n, err := decoder.Read(readBuf)
//         if(err != nil || n <= 0){
//             break
//         }

//         if n > 0 {
// 			// append to pcm buffer
// 			pcms := bytesToInt16LE(readBuf[:n])
// 			pcmBuf = append(pcmBuf, pcms...)

// 			sendPcmBytes += n
// 		}

// 		if len(pcmBuf) < samplesPerFrame {
// 			time.Sleep(5 * time.Millisecond)
// 			continue
// 		}

//         // 重采样：44100 -> 48000
//         resampled := resampleLinearInt16(pcmBuf, srcRate, outRate, channels)

//         if len(resampled) < samplesPerFrame {
// 			continue
// 		}

//         frame := resampled[:samplesPerFrame] // 960*2

// 		// ????
// 		// pcmBuf = pcmBuf[(samplesPerFrame*outRate/srcRate):] // 移除已用部分（对齐到原采样率）

// 		srcSamplesToRemove := (samplesPerFrame * srcRate) / outRate
//         pcmBuf = pcmBuf[srcSamplesToRemove:]

//         opusData := make([]byte, 4000)
//         outLen, err := enc.Encode(frame, opusData)
// 		if err != nil {
// 			fmt.Println("opus Encode oocur:", err)
// 			break
// 		}

//         <-ticker.C

//         err = conn.WriteMessage(websocket.BinaryMessage, opusData[:outLen])
// 		if err != nil {
// 			fmt.Println("ws write:", err)
// 			break
// 		}

//         fmt.Println("send opus data len",outLen ,
// 			"origin size",n,
// 			sendPcmBytes , "/", totalPcmBytes)
//     }//end for

// 	fmt.Println("mp3 opus encode end:", err)
// 	conn.WriteMessage(websocket.TextMessage, []byte("musicstop"))
// }

func AudioPlayOpus(conn *websocket.Conn) {
    fmt.Println("Auido play decoder opus...")
    conn.WriteMessage(websocket.TextMessage, []byte("music_play_opus"))
    
    // f, err := os.Open("audio/san.mp3")
    f, err := os.Open("audio/guang.mp3")
    if err != nil {
        fmt.Println("read file error", err.Error())
        return
    }

    defer f.Close()

    // 解码为 PCM
    decoder, err := mp3.NewDecoder(f)
    if err != nil {
        fmt.Println("mp3 decode error", err.Error())
        return 
    }

    srcRate := decoder.SampleRate() //原始采样率 (e.g., 44100)
    channels := 2
    fmt.Println("MP3 sample rate:", srcRate, decoder.Length())

    outRate := 48000 //opus推荐采样率 
    
    // 创建 Opus Encoder (代码不变)
    enc, err := opus.NewEncoder(outRate, channels, opus.AppAudio)
    if err != nil {
        fmt.Println("opus encode create error", err.Error())
        return
    }
    enc.SetBitrate(64000)

    // 每帧 20ms
    frameSize := outRate * 20 / 1000 
    samplesPerFrame := frameSize * channels // 1920 int16 samples

    // *** 引入缓冲上限：防止 pcmBuf 无限膨胀 ***
    // 目标: 限制 pcmBuf 缓冲最多 100ms 的数据
    // 100ms * srcRate (44100) * 2 channels = 8820 int16 样本
    const MaxPcmBufSamples = 8820 

    // 预计算：编码一帧 Opus 需要移除的原始样本数
    srcSamplesToRemove := (samplesPerFrame * srcRate) / outRate // 1764 int16 样本
    
    readBuf := make([]byte, 8 * 1024) // MP3 解码读取 buffer
    pcmBuf := make([]int16, 0) 

    ticker := time.NewTicker(20 * time.Millisecond)
    defer ticker.Stop()
    
    totalPcmBytes := decoder.Length()
    sendPcmBytes := 0
    for{
        // --- 缓冲限制逻辑 ---
        // 如果缓冲已满，暂停解码，等待发送循环消耗数据，以维持实时性
		var readed int = 0
        if len(pcmBuf) >= MaxPcmBufSamples {
            time.Sleep(5 * time.Millisecond) 
            // 注意：我们不使用 continue，因为即使不解码，我们仍需要检查并发送已缓冲的数据
        } else {
            // --- 文件解码逻辑 ---
            n, err := decoder.Read(readBuf)
			readed = n
            if err != nil && err != io.EOF && n <= 0 { 
                // 遇到非 EOF 错误时退出
                fmt.Println("MP3 Read error:", err.Error())
                break
            }
            
            if n > 0 {
                // append to pcm buffer
                pcms := bytesToInt16LE(readBuf[:n])
                pcmBuf = append(pcmBuf, pcms...)
                sendPcmBytes += n
            } else if err == io.EOF && len(pcmBuf) < srcSamplesToRemove + 1 {
                // 如果文件读完，且 pcmBuf 中剩余数据不足以构成一帧，则退出
                break 
            }
        }
        
        // --- 编码和发送逻辑 ---
        
        // 检查是否有足够的原始数据来产生一帧 Opus (srcSamplesToRemove + 1)
        if len(pcmBuf) < srcSamplesToRemove + 1 { 
            // 缓冲不足，休眠等待下一次解码读取
            time.Sleep(1 * time.Millisecond)
            continue
        }

        // 1. 截取需要被重采样的原始 PCM 数据块
        // 稳妥起见，我们送入 srcSamplesToRemove + 1 个样本进行重采样
        pcmBlock := pcmBuf[:srcSamplesToRemove + 1] 

        // 2. 对这个小块进行重采样
        resampled := resampleLinearInt16(pcmBlock, srcRate, outRate, channels)
		
        // 3. 检查重采样结果是否足够输出一帧 Opus
        if len(resampled) < samplesPerFrame { 
             // 理论上不会发生，但如果发生，跳过并继续累积
             fmt.Println("Warning: Resampled data too small:", len(resampled))
             time.Sleep(1 * time.Millisecond)
             continue
        }

        // 4. 提取 Opus 帧
        frame := resampled[:samplesPerFrame] 

        // 5. 移除已用的原始样本
        pcmBuf = pcmBuf[srcSamplesToRemove:] // 移除 1764 个原始样本

        // 6. Opus 编码
        opusData := make([]byte, 4000)
        outLen, err := enc.Encode(frame, opusData)
        if err != nil {
            fmt.Println("opus Encode oocur:", err)
            break
        }

        // 7. 等待 Ticker 并推送数据 (维持 20ms 实时速率)
        <-ticker.C

        err = conn.WriteMessage(websocket.BinaryMessage, opusData[:outLen])
        if err != nil {
            fmt.Println("ws write:", err)
            break
        }
        
        fmt.Println("send opus data len",outLen , 
            "origin size",readed, 
            sendPcmBytes , "/", totalPcmBytes,
            "pcmBuf size:", len(pcmBuf)) // 观察 pcmBuf 大小是否稳定
    }//end for
    
    fmt.Println("mp3 opus encode end:", err)
    conn.WriteMessage(websocket.TextMessage, []byte("musicstop"))
}

func resampleLinearInt16(in []int16, inRate, outRate, channels int) []int16 {
	if inRate == outRate {
		// 直接返回副本
		out := make([]int16, len(in))
		copy(out, in)
		return out
	}
	// 每声道样本数
	inFrames := len(in) / channels
	// 目标帧数 (向上取整)
	outFrames := int(float64(inFrames) * float64(outRate) / float64(inRate))

	out := make([]int16, outFrames*channels)
	// ratio := float64(inFrames-1) / float64(outFrames-1) // map output index -> input float index

	for of := range outFrames {
		// use exact mapping:
		inPos := float64(of) * float64(inFrames-1) / float64(outFrames-1)
		i0 := max(int(inPos), 0)
		i1 := i0 + 1
		if i1 >= inFrames {
			i1 = inFrames - 1
		}
		alpha := inPos - float64(i0)
		for ch := range channels {
			s0 := float64(in[i0*channels+ch])
			s1 := float64(in[i1*channels+ch])
			val := s0*(1.0-alpha) + s1*alpha
			// clamp to int16
			if val > 32767 {
				val = 32767
			}
			if val < -32768 {
				val = -32768
			}
			out[of*channels+ch] = int16(val)
		}
	}
	return out
}

func bytesToInt16LE(b []byte) []int16 {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2 : i*2+2]))
	}
	return out
}

// 将 16-bit PCM 转为 float32（Opus 编码器需要）
func Pcm16ToFloat32(src []byte) []float32 {
    samples := make([]float32, len(src)/2)
    for i := range samples {
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
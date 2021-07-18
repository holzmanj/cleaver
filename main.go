package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

const UDP_PORT = 34567
const BUF_SIZE = 16

func listenUDPMessages(c chan string) {
	pc, err := net.ListenPacket("udp", fmt.Sprintf(":%d", UDP_PORT))
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	for {
		buf := make([]byte, BUF_SIZE)
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}

		c <- string(buf[0:n])
	}
}

func main() {
	c := make(chan string)

	go listenUDPMessages(c)

	// open sound file
	f, err := os.Open("breaks/Intelligent Junglist.wav")
	// f, err := os.Open("counting.wav")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	chain := NewChain(*buffer, 8)

	for {
		cmd := <-c

		commandEffects := ParseCommand(cmd)

		speaker.Lock()
		for _, effect := range commandEffects {
			effect(chain)
		}
		speaker.Unlock()
	}
}

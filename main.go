package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
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

	sampleRate := beep.SampleRate(44100)
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))

	chains := make([]*Chain, 2)
	chains[0] = NewChain(sampleRate)
	chains[1] = NewChain(sampleRate)

	chains[0].LoadSound("samples/breaks/Intelligent Junglist.wav", 8)
	chains[1].LoadSound("samples/counting.wav", 8)

	for {
		cmd := <-c

		chainIdx, effects := ParseCommand(cmd)
		if chainIdx < 0 || chainIdx > len(chains) {
			fmt.Printf("Chain %d does not exist.\n", chainIdx)
			continue
		}

		chain := chains[chainIdx]

		speaker.Lock()
		for _, effect := range effects {
			effect(chain)
		}
		speaker.Unlock()
	}
}

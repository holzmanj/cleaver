package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

const WEBSERVER_PORT = 7775
const UDP_PORT = 7776
const BUF_SIZE = 16

var Chains []*Chain

func runUDPListener(c chan string) {
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

	go runUDPListener(c)
	go runWebServer(WEBSERVER_PORT)

	sampleRate := beep.SampleRate(44100)
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))

	Chains = make([]*Chain, 4)
	for i := range Chains {
		Chains[i] = NewChain(sampleRate)
	}

	Chains[0].LoadSound("samples/breaks/Intelligent Junglist.wav", 8)
	Chains[1].LoadSound("samples/breaks/music is so special.wav", 8)
	Chains[2].LoadSound("samples/a# strings.wav", 2)
	Chains[3].LoadSound("samples/a# bass.wav", 4)

	for {
		cmd := <-c

		chainIdx, effects := ParseCommand(cmd)
		if chainIdx < 0 || chainIdx >= len(Chains) {
			fmt.Printf("Chain %d does not exist.\n", chainIdx)
			continue
		}

		chain := Chains[chainIdx]

		speaker.Lock()
		for _, effect := range effects {
			effect(chain)
		}
		speaker.Unlock()

		PushChainConfigs()
	}
}

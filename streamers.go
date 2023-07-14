package main

import (
	"log"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

/*
STREAM CHAIN

represents one independently-playing channel
holds references to all streams in chain, but they should be composed like so:
ChoppedSound -> Resampler -> Mincer -> Pan -> Volume

*/

type Chain struct {
	Config ChainConfig
	chops  *ChoppedSound
	speed  *beep.Resampler
	mince  *Mincer
	pan    *effects.Pan
	vol    *effects.Volume
}

type ChainConfig struct {
	SoundPath     string `json:"path"`
	NumChops      int    `json:"chops"`
	SpeedN        int    `json:"speedN"`
	SpeedD        int    `json:"speedD"`
	MinceSize     int    `json:"minceSize"`
	MinceInterval int    `json:"minceInterval"`
	Pan           int    `json:"pan"`
	Volume        int    `json:"volume"`
}

func NewChain(sr beep.SampleRate) *Chain {
	ch := &ChoppedSound{sampleRate: sr, buf: nil, boundaries: make([]int, 0), activeChop: beep.Silence(-1)}
	rs := beep.ResampleRatio(4, 1.0, ch)
	mn := NewMincer(rs, sr)
	pn := &effects.Pan{Streamer: mn, Pan: 0}
	vl := &effects.Volume{Streamer: pn, Base: 2, Volume: 0, Silent: false}

	speaker.Play(vl)

	return &Chain{
		
		chops: ch,
		speed: rs,
		mince: mn,
		pan:   pn,
		vol:   vl,
	}
}

func (c *Chain) LoadSound(path string, nChops int) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	c.Config.SoundPath = path
	c.Config.NumChops = nChops

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()

	c.chops.SetSound(buffer)
	c.chops.Rechop(nChops)
}

func (c *Chain) PlayChop(i int) {
	c.chops.PlayChop(i)
}

func (c *Chain) RechopSound(nChops int) {
	c.chops.Rechop(nChops)
	c.Config.NumChops = nChops
}

func (c *Chain) SetSpeed(n, d int) {
	c.speed.SetRatio(float64(n) / float64(d))
	c.Config.SpeedN = n
	c.Config.SpeedD = d
}

func (c *Chain) Remince(size, interval int) {
	c.mince.LoadNewBuffer(size, interval)
	c.Config.MinceSize = size
	c.Config.MinceInterval = interval
}

func (c *Chain) SetPan(val int) {
	ratio := 1.0
	if val <= 32 {
		ratio = float64(val-16) / 16
	}
	c.pan.Pan = ratio
	c.Config.Pan = val
}

func (c *Chain) SetVolume(val int) {
	if val <= 0 {
		c.vol.Silent = true
	} else {
		if val >= 32 {
			c.vol.Volume = 0
		} else {
			c.vol.Volume = float64(val)*3/16 - 6
		}
		c.vol.Silent = false
	}
	c.Config.Volume = val
}

/*
CHOPPED SOUND

Keeps a buffer of the entire sound and the boundary positions of some number of chops.
*/
type ChoppedSound struct {
	sampleRate beep.SampleRate
	buf        *beep.Buffer
	boundaries []int
	activeChop beep.Streamer
}

func (cs *ChoppedSound) SetSound(buf *beep.Buffer) {
	cs.buf = buf
	cs.Rechop(1)
}

// changes the number of chops of the original sound sample
func (cs *ChoppedSound) Rechop(nChops int) {
	bounds := make([]int, nChops+1)
	var slen float32 = float32(cs.buf.Len()) / float32(nChops)
	for i := range bounds {
		bounds[i] = i * int(slen)
	}
	cs.boundaries = bounds
}

// plays the chop of index (i % num chops), cutting off any currently playing chop
func (cs *ChoppedSound) PlayChop(i int) {
	if cs.buf == nil {
		cs.activeChop = beep.Silence(-1)
		return
	}

	i = i % (len(cs.boundaries) - 1)
	start := cs.boundaries[i]
	end := cs.boundaries[i+1]

	raw := cs.buf.Streamer(start, end)
	resampled := beep.Resample(4, cs.buf.Format().SampleRate, cs.sampleRate, raw)

	cs.activeChop = resampled
}

func (cs *ChoppedSound) Stream(samples [][2]float64) (n int, ok bool) {
	filled := 0

	for filled < len(samples) {
		// stream from active chop
		n, ok := cs.activeChop.Stream(samples[filled:])

		// if active chop streamer is exhausted, just play silence
		if !ok {
			for i := range samples[filled:] {
				samples[i][0] = 0
				samples[i][1] = 0
			}
			break
		}

		filled += n
	}

	return len(samples), true
}

func (cs *ChoppedSound) Err() error {
	return cs.activeChop.Err()
}

/*
MINCER

divides audio into small repeating slices
if size is 0, just play the wrapped streamer as is
*/
type Mincer struct {
	streamer    beep.Streamer
	sampleRate  beep.SampleRate
	buf         [][2]float64
	refillBuf   bool
	bufPos      int
	intervalLen int
	intervalPos int
}

func NewMincer(streamer beep.Streamer, sr beep.SampleRate) *Mincer {
	return &Mincer{
		streamer:    streamer,
		sampleRate:  sr,
		buf:         make([][2]float64, 0),
		refillBuf:   true,
		bufPos:      0,
		intervalLen: 0,
		intervalPos: 0,
	}
}

// refills the Mincer's buffer with some number of samples from the wrapped streamer at the next Stream call
// new buffer will be (4ms * size) samples long
// interval will also be a multiple of 4ms long
func (m *Mincer) LoadNewBuffer(size, interval int) {
	m.refillBuf = true
	m.buf = make([][2]float64, (int(m.sampleRate)*size*4)/1000)
	m.intervalLen = interval * 400
}

func (m *Mincer) Stream(samples [][2]float64) (n int, ok bool) {
	if len(m.buf) == 0 || m.intervalLen == 0 {
		return m.streamer.Stream(samples)
	}

	readFromStream := 0
	if m.refillBuf {
		amt, _ := m.streamer.Stream(m.buf)
		readFromStream += amt
		m.bufPos = 0
		m.intervalPos = 0
		m.refillBuf = false
	}

	samplesFilled := 0
	for i := range samples {
		samples[i][0] = m.buf[m.bufPos][0]
		samples[i][1] = m.buf[m.bufPos][1]
		samplesFilled++

		m.bufPos = (m.bufPos + 1) % len(m.buf)

		// return early if reached end of interval
		m.intervalPos++
		if m.intervalPos >= m.intervalLen {
			m.refillBuf = true
			break
		}
	}

	// advance wrapped stream
	if diff := samplesFilled - readFromStream; diff > 0 {
		discard := make([][2]float64, diff)
		m.streamer.Stream(discard)
	}

	return samplesFilled, true
}

func (m *Mincer) Err() error {
	return nil
}

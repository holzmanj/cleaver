package main

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

/*
STREAM CHAIN

represents one independently-playing channel
holds references to all streams in chain, but they should be composed like so:
ChoppedSound -> Resampler -> Mincer -> Pan -> Volume

*/

type Chain struct {
	chops *ChoppedSound
	speed *beep.Resampler
	mince *Mincer
	pan   *effects.Pan
	vol   *effects.Volume
}

func NewChain(buf beep.Buffer, nchops int) *Chain {
	ch := NewChoppedSound(buf, nchops)
	rs := beep.ResampleRatio(4, 1.0, ch)
	mn := NewMincer(rs, buf.Format())
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

func (c *Chain) PlayChop(i int) {
	c.chops.PlayChop(i)
}

func (c *Chain) RechopSound(nChops int) {
	c.chops.Rechop(nChops)
}

func (c *Chain) SetSpeed(ratio float64) {
	c.speed.SetRatio(ratio)
}

func (c *Chain) Remince(size, interval int) {
	c.mince.LoadNewBuffer(size, interval)
}

func (c *Chain) SetPan(ratio float64) {
	c.pan.Pan = ratio
}

// set the volume of the chain
// expects a float in range [0, 1] where 0 is muted, and 1.0 is full volume
func (c *Chain) SetVolume(vol float64) {
	if vol <= 0 {
		c.vol.Silent = true
	} else {
		if vol > 1 {
			vol = 1
		}
		c.vol.Silent = false
		c.vol.Volume = -((1 - vol) * 6)
	}
}

/*
CHOPPED SOUND

Keeps a buffer of the entire sound and the boundary positions of some number of chops.
*/
type ChoppedSound struct {
	buf        beep.Buffer
	boundaries []int
	activeChop beep.Streamer
}

func NewChoppedSound(buf beep.Buffer, nChops int) *ChoppedSound {
	// generate evenly spaced chop boundaries by default
	bounds := make([]int, nChops+1)
	var slen float32 = float32(buf.Len()) / float32(nChops)
	for i := range bounds {
		bounds[i] = i * int(slen)
	}

	return &ChoppedSound{
		buf:        buf,
		boundaries: bounds,
		activeChop: beep.Silence(-1),
	}
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
	i = i % (len(cs.boundaries) - 1)
	start := cs.boundaries[i]
	end := cs.boundaries[i+1]

	cs.activeChop = cs.buf.Streamer(start, end)
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

func NewMincer(streamer beep.Streamer, format beep.Format) *Mincer {
	return &Mincer{
		streamer:    streamer,
		sampleRate:  format.SampleRate,
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

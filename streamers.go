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
ChoppedSound -> Resampler -> Pan -> Volume

*/

type Chain struct {
	chops *ChoppedSound
	speed *beep.Resampler
	pan   *effects.Pan
	vol   *effects.Volume
}

func NewChain(buf beep.Buffer, nchops int) *Chain {
	ch := NewChoppedSound(buf, nchops)
	rs := beep.ResampleRatio(4, 1.0, ch)
	pn := &effects.Pan{Streamer: rs, Pan: 0}
	vl := &effects.Volume{Streamer: pn, Base: 2, Volume: 0, Silent: false}

	speaker.Play(vl)

	return &Chain{
		chops: ch,
		speed: rs,
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

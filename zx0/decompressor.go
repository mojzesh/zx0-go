/*
 * (c) Copyright 2021 by Einar Saukas. All rights reserved.
 * (c) Copyright 2024 by Artur 'Mojzesh' Torun. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *     * Redistributions of source code must retain the above copyright
 *       notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above copyright
 *       notice, this list of conditions and the following disclaimer in the
 *       documentation and/or other materials provided with the distribution.
 *     * The name of its author may not be used to endorse or promote products
 *       derived from this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package zx0

import "fmt"

type Decompressor struct {
	lastOffset int
	inputData  []byte
	output     []byte
	inputIndex int
	bitMask    int
	bitValue   int
	backwards  bool
	inverted   bool
	backtrack  bool
	lastByte   int
}

func NewDecompressor() *Decompressor {
	return &Decompressor{}
}

func (d *Decompressor) readByte() int {
	d.lastByte = int(d.inputData[d.inputIndex])
	d.inputIndex++
	return d.lastByte
}

func (d *Decompressor) readBit() int {
	if d.backtrack {
		d.backtrack = false
		return int(d.inputData[d.inputIndex-1] & 0x01)
	}
	d.bitMask >>= 1
	if d.bitMask == 0 {
		d.bitMask = 128
		d.bitValue = d.readByte()
	}

	if (d.bitValue & d.bitMask) != 0 {
		return 1
	} else {
		return 0
	}
}

func (d *Decompressor) readInterlacedEliasGamma(msb bool) int {
	value := 1
	for d.readBit() == btoi(d.backwards) {
		value = value<<1 | d.readBit() ^ btoi(msb && d.inverted)
	}
	return value
}

func (d *Decompressor) writeByte(value int) {
	d.output = append(d.output, byte(value&0xff))
}

func (d *Decompressor) copyBytes(length int) {
	for ; length > 0; length-- {
		d.output = append(d.output, d.output[len(d.output)-d.lastOffset])
	}
}

func (d *Decompressor) Decompress(input []byte, backwardsMode, invertMode bool) ([]byte, error) {
	d.lastOffset = INITIAL_OFFSET
	d.inputData = input
	d.output = []byte{}
	d.inputIndex = 0
	d.bitMask = 0
	d.backwards = backwardsMode
	d.inverted = invertMode
	d.backtrack = false

	state := COPY_LITERALS
	for state != COPY_END {
		state = state.Process(d)
		if state == COPY_UNKNOWN {
			return nil, fmt.Errorf("Decompression error: invalid state")
		}
	}
	return d.output, nil
}

type State int

const (
	COPY_LITERALS State = iota
	COPY_FROM_LAST_OFFSET
	COPY_FROM_NEW_OFFSET
	COPY_END
	COPY_UNKNOWN
)

func (s State) Process(d *Decompressor) State {
	switch s {
	case COPY_LITERALS:
		length := d.readInterlacedEliasGamma(false)
		for i := 0; i < length; i++ {
			d.writeByte(d.readByte())
		}
		if d.readBit() == 0 {
			return COPY_FROM_LAST_OFFSET
		}
		return COPY_FROM_NEW_OFFSET
	case COPY_FROM_LAST_OFFSET:
		length := d.readInterlacedEliasGamma(false)
		d.copyBytes(length)
		if d.readBit() == 0 {
			return COPY_LITERALS
		}
		return COPY_FROM_NEW_OFFSET
	case COPY_FROM_NEW_OFFSET:
		msb := d.readInterlacedEliasGamma(true)
		if msb == 256 {
			return COPY_END
		}
		lsb := d.readByte() >> 1
		if d.backwards {
			d.lastOffset = (msb*128 + lsb - 127)
		} else {
			d.lastOffset = (msb*128 - lsb)
		}
		d.backtrack = true
		length := d.readInterlacedEliasGamma(false) + 1
		d.copyBytes(length)
		if d.readBit() == 0 {
			return COPY_LITERALS
		}
		return COPY_FROM_NEW_OFFSET
	default:
		return COPY_UNKNOWN
	}
}

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

type Compressor struct {
	output      []byte
	outputIndex int
	inputIndex  int
	bitIndex    int
	bitMask     int
	diff        int
	backtrack   bool
}

func NewCompressor() *Compressor {
	return &Compressor{}
}

func (c *Compressor) readBytes(n int, delta []int) {
	c.inputIndex += n
	c.diff += n
	if delta[0] < c.diff {
		delta[0] = c.diff
	}
}

func (c *Compressor) writeByte(value int) {
	c.output[c.outputIndex] = byte(value & 0xff)
	c.outputIndex++
	c.diff--
}

func (c *Compressor) writeBit(value int) {
	if c.backtrack {
		if value > 0 {
			c.output[c.outputIndex-1] |= 1
		}
		c.backtrack = false
	} else {
		if c.bitMask == 0 {
			c.bitMask = 128
			c.bitIndex = c.outputIndex
			c.writeByte(0)
		}
		if value > 0 {
			c.output[c.bitIndex] |= byte(c.bitMask)
		}
		c.bitMask >>= 1
	}
}

func (c *Compressor) writeInterlacedEliasGamma(value int, backwardsMode, invertMode bool) {
	i := 2
	for i <= value {
		i <<= 1
	}
	i >>= 1
	for i >>= 1; i > 0; i >>= 1 {
		c.writeBit(btoi(backwardsMode))
		c.writeBit(btoi(invertMode == ((value & i) == 0)))
	}
	c.writeBit(btoi(!backwardsMode))
}

func (c *Compressor) Compress(optimal *Block, input []byte, skip int, backwardsMode, invertMode bool, delta []int) []byte {
	lastOffset := INITIAL_OFFSET

	// calculate and allocate output buffer
	c.output = make([]byte, (optimal.Bits+25)/8)

	// un-reverse optimal sequence
	var prev *Block
	for optimal != nil {
		next := optimal.Chain
		optimal.Chain = prev
		prev = optimal
		optimal = next
	}

	// initialize data
	c.diff = len(c.output) - len(input) + skip
	delta[0] = 0
	c.inputIndex = skip
	c.outputIndex = 0
	c.bitMask = 0
	c.backtrack = true

	// generate output
	for optimal = prev.Chain; optimal != nil; prev, optimal = optimal, optimal.Chain {
		length := optimal.Index - prev.Index
		if optimal.Offset == 0 {
			// copy literals indicator
			c.writeBit(0)

			// copy literals length
			c.writeInterlacedEliasGamma(length, backwardsMode, false)

			// copy literals values
			for i := 0; i < length; i++ {
				c.writeByte(int(input[c.inputIndex]))
				c.readBytes(1, delta)
			}
		} else if optimal.Offset == lastOffset {
			// copy from last offset indicator
			c.writeBit(0)

			// copy from last offset length
			c.writeInterlacedEliasGamma(length, backwardsMode, false)
			c.readBytes(length, delta)
		} else {
			// copy from new offset indicator
			c.writeBit(1)

			// copy from new offset MSB
			c.writeInterlacedEliasGamma((optimal.Offset-1)/128+1, backwardsMode, invertMode)

			// copy from new offset LSB
			c.writeByte(btoi(backwardsMode)*((optimal.Offset-1)%128)<<1 + btoi(!backwardsMode)*(127-(optimal.Offset-1)%128)<<1)

			// copy from new offset length
			c.backtrack = true
			c.writeInterlacedEliasGamma(length-1, backwardsMode, false)
			c.readBytes(length, delta)

			lastOffset = optimal.Offset
		}
	}

	// end marker
	c.writeBit(1)
	c.writeInterlacedEliasGamma(256, backwardsMode, invertMode)

	// done!
	return c.output
}

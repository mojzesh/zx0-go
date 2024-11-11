package zx0

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
)

const (
	INITIAL_OFFSET = 1
	MAX_SCALE      = 50
)

type Optimizer struct {
	lastLiteral []*Block
	lastMatch   []*Block
	optimal     []*Block
	matchLength []int
	bestLength  []int
}

func NewOptimizer() *Optimizer {
	return &Optimizer{}
}

func offsetCeiling(index, offsetLimit int) int {
	return min(max(index, INITIAL_OFFSET), offsetLimit)
}

func eliasGammaBits(value int) int {
	bits := 1
	for value > 1 {
		bits += 2
		value >>= 1
	}
	return bits
}

type Job struct {
	initialOffset, finalOffset, index, skip int
}

type JobResult struct {
	Block         *Block
	initialOffset int
}

func (o *Optimizer) Optimize(input []byte, skip, offsetLimit, threads int, verbose bool) *Block {
	arraySize := offsetCeiling(len(input)-1, offsetLimit) + 1
	o.lastLiteral = make([]*Block, arraySize)
	o.lastMatch = make([]*Block, arraySize)
	o.optimal = make([]*Block, len(input))
	o.matchLength = make([]int, arraySize)
	o.bestLength = make([]int, len(input))
	if len(o.bestLength) > 2 {
		o.bestLength[2] = 2
	}

	o.lastMatch[INITIAL_OFFSET] = &Block{-1, skip - 1, INITIAL_OFFSET, nil}

	if threads <= 0 {
		threads = runtime.NumCPU()
	}
	fmt.Printf("Using: %d thread(s)\n", threads)

	dots := 2
	if verbose {
		fmt.Print("[")
	}

	if threads == 1 {
		for index := skip; index < len(input); index++ {
			maxOffset := offsetCeiling(index, offsetLimit)
			o.optimal[index] = o.processTask(1, maxOffset, index, skip, input)
			if verbose && index*MAX_SCALE/len(input) > dots {
				fmt.Print(".")
				dots++
			}
		}
	} else {
		for index := skip; index < len(input); index++ {
			maxOffset := offsetCeiling(index, offsetLimit)
			taskSize := maxOffset/threads + 1

			inputJobsChan := make(chan *Job, threads)
			outputTaskChan := make(chan *JobResult, threads)

			var wgSend sync.WaitGroup
			for i := 0; i < threads; i++ {
				wgSend.Add(1)
				go worker(outputTaskChan, inputJobsChan, &wgSend, o, input)
			}

			var wgRecv sync.WaitGroup
			wgRecv.Add(1)
			go func(index int) {
				defer wgRecv.Done()
				// Collect results out of order
				results := []*JobResult{}
				for jobResult := range outputTaskChan {
					if jobResult.Block != nil {
						results = append(results, jobResult)
						if verbose && index*MAX_SCALE/len(input) > dots {
							fmt.Print(".")
							dots++
						}
					}
				}

				// Sort results by initialOffset
				sort.Slice(results, func(i, j int) bool {
					return results[i].initialOffset < results[j].initialOffset
				})

				// Find optimal block
				for _, result := range results {
					if o.optimal[index] == nil || o.optimal[index].Bits > result.Block.Bits {
						o.optimal[index] = result.Block
					}
				}
			}(index)

			for initialOffset := 1; initialOffset <= maxOffset; initialOffset += taskSize {
				finalOffset := min(initialOffset+taskSize-1, maxOffset)
				inputJobsChan <- &Job{initialOffset, finalOffset, index, skip}
			}

			close(inputJobsChan)
			wgSend.Wait()

			close(outputTaskChan)
			wgRecv.Wait()
		}

	}

	if verbose {
		fmt.Println("]")
	}

	return o.optimal[len(input)-1]
}

func worker(outputTaskChan chan *JobResult, inputJobsChan chan *Job, wgSend *sync.WaitGroup, o *Optimizer, input []byte) {
	defer wgSend.Done()
	for inputJob := range inputJobsChan {
		outputTaskChan <- &JobResult{
			Block:         o.processTask(inputJob.initialOffset, inputJob.finalOffset, inputJob.index, inputJob.skip, input),
			initialOffset: inputJob.initialOffset,
		}
	}
}

func (o *Optimizer) processTask(initialOffset, finalOffset, index, skip int, input []byte) *Block {
	bestLengthSize := 2
	var optimalBlock *Block
	for offset := initialOffset; offset <= finalOffset; offset++ {
		if index != skip && index >= offset && input[index] == input[index-offset] {
			if o.lastLiteral[offset] != nil {
				length := index - o.lastLiteral[offset].Index
				bits := o.lastLiteral[offset].Bits + 1 + eliasGammaBits(length)
				o.lastMatch[offset] = &Block{bits, index, offset, o.lastLiteral[offset]}
				if optimalBlock == nil || optimalBlock.Bits > bits {
					optimalBlock = o.lastMatch[offset]
				}
			}
			if o.matchLength[offset]++; o.matchLength[offset] > 1 {
				if bestLengthSize < o.matchLength[offset] {
					bits := o.optimal[index-o.bestLength[bestLengthSize]].Bits + eliasGammaBits(o.bestLength[bestLengthSize]-1)
					for {
						bestLengthSize++
						bits2 := o.optimal[index-bestLengthSize].Bits + eliasGammaBits(bestLengthSize-1)
						if bits2 <= bits {
							o.bestLength[bestLengthSize] = bestLengthSize
							bits = bits2
						} else {
							o.bestLength[bestLengthSize] = o.bestLength[bestLengthSize-1]
						}
						if !(bestLengthSize < o.matchLength[offset]) {
							break
						}
					}
				}
				length := o.bestLength[o.matchLength[offset]]
				bits := o.optimal[index-length].Bits + 8 + eliasGammaBits((offset-1)/128+1) + eliasGammaBits(length-1)
				if o.lastMatch[offset] == nil || o.lastMatch[offset].Index != index || o.lastMatch[offset].Bits > bits {
					o.lastMatch[offset] = &Block{bits, index, offset, o.optimal[index-length]}
					if optimalBlock == nil || optimalBlock.Bits > bits {
						optimalBlock = o.lastMatch[offset]
					}
				}
			}
		} else {
			o.matchLength[offset] = 0
			if o.lastMatch[offset] != nil {
				length := index - o.lastMatch[offset].Index
				bits := o.lastMatch[offset].Bits + 1 + eliasGammaBits(length) + length*8
				o.lastLiteral[offset] = &Block{bits, index, 0, o.lastMatch[offset]}
				if optimalBlock == nil || optimalBlock.Bits > bits {
					optimalBlock = o.lastLiteral[offset]
				}
			}
		}
	}

	return optimalBlock
}

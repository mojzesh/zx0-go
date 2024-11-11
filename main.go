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

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/mojzesh/zx0-go/zx0"
)

const (
	MAX_OFFSET_ZX0  = 32640
	MAX_OFFSET_ZX7  = 2176
	DEFAULT_THREADS = 4
)

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func reverse(array []byte) {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}
}

func zx0Fn(input []byte, skip int, backwardsMode, classicMode, quickMode bool, threads int, verbose bool, delta []int) []byte {
	var mode int
	if quickMode {
		mode = MAX_OFFSET_ZX7
	} else {
		mode = MAX_OFFSET_ZX0
	}
	return zx0.NewCompressor().Compress(
		zx0.NewOptimizer().Optimize(input, skip, mode, threads, verbose),
		input, skip, backwardsMode, !classicMode && !backwardsMode, delta)
}

func dzx0Fn(input []byte, backwardsMode, classicMode bool) ([]byte, error) {
	return zx0.NewDecompressor().Decompress(input, backwardsMode, !classicMode && !backwardsMode)
}

func main() {
	fmt.Println("ZX0 v2.2: Optimal data compressor by Einar Saukas")
	fmt.Println("Ported to Go by Artur 'Mojzesh' Torun")

	// process optional parameters
	var threads int
	var forcedMode, classicMode, backwardsMode, quickMode, decompress bool
	var skip int

	flag.IntVar(&threads, "p", DEFAULT_THREADS, "Parallel processing with N threads, if p <= 0\nthen all available CPUs are used")
	flag.BoolVar(&forcedMode, "f", false, "Force overwrite of output file")
	flag.BoolVar(&classicMode, "c", false, "Classic file format (v1.*)")
	flag.BoolVar(&backwardsMode, "b", false, "Compress backwards")
	flag.BoolVar(&quickMode, "q", false, "Quick non-optimal compression")
	flag.BoolVar(&decompress, "d", false, "Decompress")
	flag.IntVar(&skip, "s", 0, "Skip N bytes")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || len(args) > 2 {
		fmt.Println("Usage: zx0 [-pN] [-f] [-c] [-b] [-q] [-d] input [output.zx0]")
		os.Exit(1)
	}

	if decompress && skip > 0 {
		fmt.Println("Error: Decompressing with suffix not supported")
		os.Exit(1)
	}

	// determine output filename
	var outputName string
	if len(args) == 1 {
		if !decompress {
			outputName = args[0] + ".zx0"
		} else {
			if len(args[0]) > 4 && args[0][len(args[0])-4:] == ".zx0" {
				outputName = args[0][:len(args[0])-4]
			} else {
				fmt.Println("Error: Cannot infer output filename")
				os.Exit(1)
			}
		}
	} else {
		outputName = args[1]
	}

	// read input file
	input, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Printf("Error: Cannot read input file %s\n", args[0])
		os.Exit(1)
	}

	// determine input size
	if len(input) == 0 {
		fmt.Printf("Error: Empty input file %s\n", args[0])
		os.Exit(1)
	}

	// validate skip against input size
	if skip >= len(input) {
		fmt.Printf("Error: Skipping entire input file %s\n", args[0])
		os.Exit(1)
	}

	// check output file
	if !forcedMode && fileExists(outputName) {
		fmt.Printf("Error: Already existing output file %s\n", outputName)
		os.Exit(1)
	}

	// conditionally reverse input file
	if backwardsMode {
		reverse(input)
	}

	// generate output file
	var output []byte
	delta := []int{0}

	if !decompress {
		output = zx0Fn(input, skip, backwardsMode, classicMode, quickMode, threads, true, delta)
	} else {
		output, err = dzx0Fn(input, backwardsMode, classicMode)
		if err != nil {
			fmt.Printf("Error: Invalid input file %s\n", args[0])
			os.Exit(1)
		}
	}

	// conditionally reverse output file
	if backwardsMode {
		reverse(output)
	}

	// write output file
	err = os.WriteFile(outputName, output, 0644)
	if err != nil {
		fmt.Printf("Error: Cannot write output file %s\n", outputName)
		os.Exit(1)
	}

	var backwardsModeStr string
	if backwardsMode {
		backwardsModeStr = "backwards "
	} else {
		backwardsModeStr = ""
	}

	// done!
	if !decompress {
		var compTypeStr string
		if skip > 0 {
			compTypeStr = "partially "
		} else {
			compTypeStr = ""
		}
		fmt.Printf("File %scompressed %sfrom %d to %d bytes! (delta %d)\n",
			compTypeStr,
			backwardsModeStr,
			len(input)-skip, len(output), delta[0])
	} else {
		fmt.Printf("File decompressed %sfrom %d to %d bytes!\n",
			backwardsModeStr,
			len(input)-skip, len(output))
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

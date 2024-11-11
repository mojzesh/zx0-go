# zx0-go

**zx0-go** is a multi-thread implementation of the
[ZX0](https://github.com/einar-saukas/ZX0) data compressor in
[Go](https://golang.org/).


## Requirements

To run this compressor, you must have installed [Go](https://golang.org/) 1.23
or later.


## Usage

To compress a file such as "Cobra.scr", use the command-line compressor as
follows:

```
go run main.go Cobra.scr
```

This compressor uses 4 threads by default. You can use parameter "-p" to
specify a different number of threads, for instance:

```
go run main.go -p=2 Cobra.scr
```

If number of threads is set to 0 or negative, the compressor will use the
maximum number of threads equal to count of available CPUs in the system.

All other parameters work exactly like the original version. Check the official
[ZX0](https://github.com/einar-saukas/ZX0) page for further details.


## Building

To build the compressor binary, you can use the following command:

```
go build -o ./bin/zx0 .
```

or use the makefile:

```
make build
```

output binary will be placed in the "bin" directory.

## License

The Go implementation of [ZX0](https://github.com/mojzesh/zx0-go) was
authored by **Artur 'Mojzesh' Torun** and it's available under the "BSD-3" license.


## Links

* [ZX0](https://github.com/einar-saukas/ZX0) - The original version of **ZX0**,
by Einar Saukas.

* [ZX0-Kotlin](https://github.com/einar-saukas/ZX0-Kotlin) - A similar
multi-thread data compressor for [ZX0](https://github.com/einar-saukas/ZX0)
in [Kotlin](https://kotlinlang.org/), by the same author.

* [ZX5-Kotlin](https://github.com/einar-saukas/ZX5-Kotlin) - A similar
multi-thread data compressor for [ZX5](https://github.com/einar-saukas/ZX5)
in [Kotlin](https://kotlinlang.org/), by the same author.

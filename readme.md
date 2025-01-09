# MP3Cat

[1]: https://www.dmulholl.com/dev/mp3cat.html
[2]: https://github.com/dmulholl/mp3cat/releases
[3]: https://github.com/crra/mp3binder


MP3Cat is a simple command line utility for concatenating MP3 files without re-encoding.

<p align="center">
    <img src="mp3cat.png" width="600px">
</p>



## Download

You can download a pre-compiled binary from the [releases page][2].



## Build

MP3Cat is written in Go. If you have a Go compiler installed you can run:

    go install github.com/dmulholl/mp3cat@latest

This will download, compile, and install the latest version of the application
to your `$GOPATH/bin` directory.



## Usage

Run `mp3cat --help` to view the command line help:

    Usage: mp3cat [files]

      This tool concatenates MP3 files without re-encoding.

    Arguments:
      [files]                 List of files to merge.

    Options:
      -d, --dir <path>        Directory of files to merge.
      -o, --out <path>        Output filepath.

    Flags:
      -f, --force             Overwrite an existing output file.
      -h, --help              Display this help text and exit.
      -q, --quiet             Run in quiet mode.
      -v, --version           Display the version number.

You can specify the input as a list of filenames, e.g.

    $ mp3cat one.mp3 two.mp3 three.mp3

Or, using globbing, e.g.

    $ mp3cat *.mp3

Alternatively, you can specify an entire directory of `.mp3` files to concatenate, e.g.

    $ mp3cat --dir /path/to/directory

The output filename defaults to `output.mp3` but you can specify a custom filename using the `-o/--out` option, e.g.

    $ mp3cat *.mp3 -o joined.mp3

This utility works most reliably with constant-bit-rate input &mdash; it will add a VBR header if it detects multiple bit-rates in the input files but variable-bit-rate MP3s have patchy player support and should generally be avoided if at all possible.



## Status

This application is in *maintenance mode* &mdash; it's intentionally simple and I'm not planning to add any extra features.
If you need something more complex check out [mp3binder][3] which is an active fork with additional functionality.



## License

This work has been placed in the public domain.

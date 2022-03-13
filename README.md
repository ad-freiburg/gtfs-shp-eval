# gtfs-shp-eval

Evaluate the quality of large sets of GTFS feeds.

## 1. Installation
    $ go get github.com/patrickbr/gtfstidy

## 2. Usage
See

    $ gtfs-shp-eval --help

for possible options.

To run an evaluation on all feeds contained in `<folder>`, use

	$ gtfs-shp-eval -v <folder>

Multiple folders can be provided. Stats are printed to stdout.

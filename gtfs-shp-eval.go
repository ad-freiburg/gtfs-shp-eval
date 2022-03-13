// Copyright 2020, University of Freiburg
// Chair of Algorithms and Data Strcutures
// Authors: Patrick Brosi <brosi@informatik.uni-freiburg.de>

package main

import (
	"fmt"
	"github.com/patrickbr/gtfsparser"
	gtfs "github.com/patrickbr/gtfsparser/gtfs"
	flag "github.com/spf13/pflag"
	"math"
	"os"
	"path/filepath"
)

var DEG_TO_RAD float64 = 0.017453292519943295769236907684886127134428718885417254560
var SHP_CACHE map[*gtfs.Shape][][]float64

func latLngToWebMerc(lat float32, lng float32) (float64, float64) {
	x := 6378137.0 * lng * float32(DEG_TO_RAD)
	a := float64(lat * float32(DEG_TO_RAD))

	lng = x
	lat = float32(3189068.5 * math.Log((1.0+math.Sin(a))/(1.0-math.Sin(a))))
	return float64(lng), float64(lat)
}

func perpDist(px, py, lax, lay, lbx, lby float64) float64 {
	d := dist(lax, lay, lbx, lby) * dist(lax, lay, lbx, lby)

	if d == 0 {
		return dist(px, py, lax, lay)
	}
	t := float64((px-lax)*(lbx-lax)+(py-lay)*(lby-lay)) / d
	if t < 0 {
		return dist(px, py, lax, lay)
	} else if t > 1 {
		return dist(px, py, lbx, lby)
	}

	return dist(px, py, lax+t*(lbx-lax), lay+t*(lby-lay))
}

func dist(x1 float64, y1 float64, x2 float64, y2 float64) float64 {
	return math.Sqrt(float64((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1)))
}

func check_shape(trip *gtfs.Trip, feed *gtfsparser.Feed, maxDist float64) bool {
	shp := trip.Shape

	if _, ok := SHP_CACHE[shp]; !ok {
		for _, p := range shp.Points {
			x, y := latLngToWebMerc(p.Lat, p.Lon)
			SHP_CACHE[shp] = append(SHP_CACHE[shp], []float64{x, y})
		}
	}

	for _, s := range trip.StopTimes {
		x, y := latLngToWebMerc(s.Stop.Lat, s.Stop.Lon)

		pepdist := math.Inf(1)

		for i := 1; i < len(SHP_CACHE[shp]); i++ {
			curdist := perpDist(x, y, SHP_CACHE[shp][i-1][0], SHP_CACHE[shp][i-1][1], SHP_CACHE[shp][i][0], SHP_CACHE[shp][i][1])
			if curdist < pepdist {
				pepdist = curdist
			}
		}

		if pepdist*math.Cos(float64(s.Stop.Lat)*DEG_TO_RAD) > maxDist {
			return false
		}
	}

	return true
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gtfs-shp-eval - (C) 2020 University of Freiburg, Chair of Algorithms and Data Structures\n\nAnalyze shape.txt quality and coverage of GTFS feeds.\n\nUsage:\n\n  %s [<options>] <folder containing input GTFS feeds>*\n\nAllowed options:\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	maxDist := flag.Float64P("max-dist", "d", 250, "max distance from station to shape")
	help := flag.BoolP("help", "?", false, "this message")

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	folders := flag.Args()
	gtfsPaths := make([]string, 0)

	for _, folder := range folders {
		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			gtfsPaths = append(gtfsPaths, path)
			return nil
		})
	}

	if len(gtfsPaths) == 0 {
		fmt.Fprintln(os.Stderr, "No GTFS location specified, see --help")
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "Error:", r)
		}
	}()

	num_trips := 0
	num_ok := 0
	num_err := 0
	num_deg := 0
	num_no_shp := 0
	num_feeds := 0
	num_feeds_w_shps := 0

	for _, gtfsPath := range gtfsPaths {
		SHP_CACHE = make(map[*gtfs.Shape][][]float64)

		loc_feed := gtfsparser.NewFeed()
		opts := gtfsparser.ParseOptions{UseDefValueOnError: true, DropErroneous: true, CheckNullCoordinates: false, EmptyStringRepl: "", ZipFix: true}
		loc_feed.SetParseOpts(opts)

		fmt.Fprintf(os.Stdout, "Parsing GTFS feed in '%s' ...", gtfsPath)
		e := loc_feed.Parse(gtfsPath)

		if e != nil {
			fmt.Fprintf(os.Stderr, "\nError while parsing GTFS feed:\n")
			fmt.Fprintln(os.Stderr, e.Error())
			fmt.Fprintf(os.Stderr, "Skipping...\n")
			continue
		}
		fmt.Fprintf(os.Stdout, " done.\n")
		num_feeds += 1

		num_trips += len(loc_feed.Trips)

		if len(loc_feed.Shapes) > 0 {
			num_feeds_w_shps += 1
		}

		for _, trip := range loc_feed.Trips {
			if trip.Shape == nil {
				num_no_shp += 1
			} else if len(trip.Shape.Points) == len(trip.StopTimes) {
				num_deg += 1
			} else if !check_shape(trip, loc_feed, *maxDist) {
				num_err += 1
			} else {
				num_ok += 1
			}
		}
	}

	fmt.Fprintf(os.Stdout, "\nAnalyzed %d feeds with %d trips\n", num_feeds, num_trips)
	fmt.Fprintf(os.Stdout, "\n%d feeds had shapes (%.2f %%)\n", num_feeds_w_shps, float64(num_feeds_w_shps)/float64(num_trips))
	fmt.Fprintf(os.Stdout, "\n%d trips with OK shape (%.2f %%), %d trips with suspicious shapes (%.2f %%), %d trips with degenerated shapes (%.2f %%), %d trips with no shapes (%.2f %%)\n", num_ok, float64(num_ok)/float64(num_trips)*100.0, num_err, float64(num_err)/float64(num_trips)*100.0, num_deg, float64(num_deg)/float64(num_trips)*100.0, num_no_shp, float64(num_no_shp)/float64(num_trips)*100.0)
}

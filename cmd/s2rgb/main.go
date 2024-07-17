package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/airbusgeo/godal"
)

// Quick&dirty way to calc percentiles from bands
type Table struct {
	count [65535]int
	sum   int
}

func (t *Table) Add(val uint16) {
	t.count[val]++
	t.sum++
}

func (t *Table) Reset() {
	t.count = [65535]int{}
	t.sum = 0
}

func (t *Table) Percentile(p float64) uint16 {
	var cumsum int = 0
	var value int = int(p * float64(t.sum))
	var i uint16
	for i = 0; i < 65535; i++ {
		cumsum += t.count[i]
		if cumsum >= value {
			break
		}
	}
	return i
}

func Percentiles(b *godal.Band, lowP float64, uppP float64) (uint16, uint16) {
	structure := b.Structure()
	readBuf := make([]uint16, structure.BlockSizeX*structure.BlockSizeY)
	var bt Table
	for block, ok := structure.FirstBlock(), true; ok; block, ok = block.Next() {
		b.Read(block.X0, block.Y0, readBuf, block.W, block.H)

		for pix := 0; pix < block.W*block.H; pix++ {
			bt.Add(readBuf[pix])
		}
	}
	return bt.Percentile(lowP), bt.Percentile(uppP)
}

func Minmax(b *godal.Band) (uint16, uint16) {
	var stats godal.Statistics
	var flag bool
	if stats, flag, _ = b.GetStatistics(); !flag {
		stats, _ = b.ComputeStatistics()
	}
	return uint16(stats.Min), uint16(stats.Max)
}

func SDevs(b *godal.Band, sdevs float64) (uint16, uint16) {
	var stats godal.Statistics
	var flag bool
	if stats, flag, _ = b.GetStatistics(); !flag {
		stats, _ = b.ComputeStatistics()
	}
	return uint16(stats.Mean - sdevs*stats.Std), uint16(stats.Mean + sdevs*stats.Std)
}

type multiString struct {
	choice string
	opts   []string
}

func (ms *multiString) String() string {
	if ms.choice != "" {
		return ms.choice
	}
	return ""
}

func (ms *multiString) Set(s string) error {
	var flag = false
	for _, opt := range ms.opts {
		if s == opt {
			flag = true
			ms.choice = s
		}
	}
	if !flag {
		return errors.New("option not found")
	}
	return nil
}

var Usage = func() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\t%s [OPTIONS] <InFilename> <OutFilename>\n", os.Args[0])
	fmt.Printf("\nOptions:\n")
	fg := flag.CommandLine.Lookup("method")
	fmt.Printf("\t-%s string (Default: %s)\n", fg.Name, fg.DefValue)
	fmt.Printf("\t\t%s\n", fg.Usage)
	order := []string{"lower", "upper", "sdevs"}
	for _, name := range order {
		fg = flag.CommandLine.Lookup(name)
		fmt.Printf("\t-%s float (Default: %s)\n", fg.Name, fg.DefValue)
		fmt.Printf("\t\t%s\n", fg.Usage)
	}
}

var (
	lowP        float64
	uppP        float64
	infilename  string
	outfilename string
	method      multiString
	sdevs       float64
)

func init() {
	flag.Usage = Usage
	method = multiString{"percentiles", []string{"percentiles", "sdevs", "minmax"}}
	flag.Var(&method, "method", `Available methods are "percentiles" "sdevs" or "minmax"`)
	flag.Float64Var(&lowP, "lower", 0.02, `Lower percentile (to be used with "percentiles" method)`)
	flag.Float64Var(&uppP, "upper", 0.98, `Upper percentile (to be used with "percentiles" method)`)
	flag.Float64Var(&sdevs, "sdevs", 1.96, `Number of standard deviations from the mean (to be used with "sdevs" method)`)
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	infilename = flag.Args()[0]
	outfilename = flag.Args()[1]
}

func main() {

	godal.RegisterAll()
	dt, err := godal.Open(infilename)
	if err != nil {
		panic(err)
	}

	// Get the subdatasets
	dm := godal.Domain("SUBDATASETS")

	sdt_fname := dt.Metadata("SUBDATASET_1_NAME", dm)

	sdt, err := godal.Open(sdt_fname)
	if err != nil {
		panic(err)
	}

	structure := sdt.Structure()
	fmt.Printf("Size of image is %vx%v\n", structure.SizeX, structure.SizeY)

	gt, _ := sdt.GeoTransform()
	pj := sdt.Projection()

	ncol, nrow := structure.SizeX, structure.SizeY
	blockW, blockH := structure.BlockSizeX, structure.BlockSizeY

	// Output datset
	odt, _ := godal.Create(godal.GTiff, outfilename, 3, godal.Byte, ncol, nrow)
	odt.SetGeoTransform(gt)
	odt.SetProjection(pj)
	defer odt.Close()

	bands := [3]int{0, 1, 2}
	var inBand godal.Band
	var outBand godal.Band

	readBuf := make([]uint16, blockW*blockH)
	writeBuf := make([]uint8, blockW*blockH)
	val := 1.0

	fmt.Printf("Calculating band minimum and maximm values...\n")

	var lower [3]uint16
	var upper [3]uint16
	// read form bands B2, B3 and B4, respectively Reg, Green and Blue
	for b := 0; b < 3; b++ {

		inBand = sdt.Bands()[bands[b]]
		switch method.choice {
		case "percentiles":
			lower[b], upper[b] = Percentiles(&inBand, lowP, uppP)
		case "sdevs":
			lower[b], upper[b] = SDevs(&inBand, sdevs)
		case "minmax":
			lower[b], upper[b] = Minmax(&inBand)
		}
		fmt.Printf("Band %v: Lower value is %v and upper value is %v\n", b, lower[b], upper[b])

	}

	fmt.Printf("Rescaling bands to RGB...\n")
	for block, ok := structure.FirstBlock(), true; ok; block, ok = block.Next() {

		// read form bands B2, B3 and B4, respectively Reg, Green and Blue
		for b := 0; b < 3; b++ {
			inBand = sdt.Bands()[bands[b]]
			outBand = odt.Bands()[b]

			inBand.Read(block.X0, block.Y0, readBuf, block.W, block.H)

			for pix := 0; pix < block.W*block.H; pix++ {
				val = float64(int(readBuf[pix])-int(lower[b])) / float64(upper[b]-lower[b])
				if val < 0 {
					val = 0
				} else if val > 1 {
					val = 1
				}
				writeBuf[pix] = uint8(val * 255)
			}
			outBand.Write(block.X0, block.Y0, writeBuf, block.W, block.H)
		}
	}
	fmt.Printf("Done!\n")
}

package main

import (
	"fmt"
	"os"
	"log"
	"bufio"
	"encoding/binary"
	"bytes"
	"math"
)

// Code taken from pa-m/numgo.
// linspace(start, stop, num=50, endpoint=True, retstep=False, dtype=None)[source]
func Linspace(start, stop float64, num int32, endPoint bool) []float64 {
	step := 0.
	if endPoint {
		if num == 1 {
			return []float64{start}
		}
		step = (stop - start) / float64(num-1)
	} else {
		if num == 0 {
			return []float64{}
		}
		step = (stop - start) / float64(num)
	}
	r := make([]float64, num, num)
	for i := 0; i < int(num); i++ {
		r[i] = start + float64(i)*step
	}
	return r
}

func ReadBIN(filename string) ([]byte, int64, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return nil, 0, statsErr
	}

	var size int64 = stats.Size()
	content := make([]byte, size)

	bufr := bufio.NewReader(file)
	_, err = bufr.Read(content)

	return content, size, err
}

func main() {
	// open file
	content, _, err := ReadBIN("RAMAN.SPC")
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}

	// read first 2 bytes to decide version
	r := bytes.NewReader(content[0:2])
	var leader struct {
		Ftflg uint8
		Fversn uint8
	}
	if err := binary.Read(r, binary.LittleEndian, &leader); err != nil {
		fmt.Println("binary.Read failed:", err)
		os.Exit(3)
	}

	if leader.Fversn == 75 {
		fmt.Printf("SPC file version 75.\n")

		// read rest of header
		r := bytes.NewReader(content[2:512])
		var header struct {
        	Fexper uint8
        	Fexp uint8
        	Fnpts int32
        	Ffirst float64
        	Flast float64
        	Fnsub int32
        	Fxtype uint8
        	Fytype uint8
        	Fztype uint8
        	Fpost uint8
        	Fdate int32 
        	Fres [9]byte
        	Fsource [9]byte
        	Fpeakpt int16
        	Fspare [32]byte 
        	Fcmnt [130]byte
        	Fcatxt [30]byte
        	Flogoff int32
        	Fmods int32
        	Fprocs uint8
        	Flevel uint8
        	Fsampin int16
        	Ffactor float32
        	Fmethod [48]byte
        	Fzinc float32
        	Fwplanes int32
        	Fwinc float32
        	Fwtype uint8
        	Freserv [187]byte
		}
		if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
			fmt.Println("binary.Read failed:", err)
			os.Exit(3)
		}

		// convert date into date struct
		var date struct {
			Year int32
			Month uint8
			Day uint8
			Hour uint8
			Minute uint8
		}
		date.Year = (header.Fdate >> 20)
		date.Month = uint8((header.Fdate >> 16) % int32(math.Pow(2, 4)))
		date.Day = uint8((header.Fdate >> 11) % int32(math.Pow(2, 5)))
		date.Hour = uint8((header.Fdate >> 6) % int32(math.Pow(2, 5)))
		date.Minute = uint8(header.Fdate % int32(math.Pow(2, 6)))

		// parse flag into booleans
		var flags struct {
			Tsprec bool
        	Tcgram bool
        	Tmulti bool
        	Trandm bool
        	Tordrd bool
        	Talabs bool
        	Txyxys bool
        	Txvals bool
		}
		flags.Tsprec = (leader.Ftflg >> 0) & 1 == 1
		flags.Tcgram = (leader.Ftflg >> 1) & 1 == 1
		flags.Tmulti = (leader.Ftflg >> 2) & 1 == 1
		flags.Trandm = (leader.Ftflg >> 3) & 1 == 1
		flags.Tordrd = (leader.Ftflg >> 4) & 1 == 1
		flags.Talabs = (leader.Ftflg >> 5) & 1 == 1
		flags.Txyxys = (leader.Ftflg >> 6) & 1 == 1
        flags.Txvals = (leader.Ftflg >> 7) & 1 == 1
		
		// multi or single spc file
		var dat_multi bool
		if header.Fnsub > 1 {
			dat_multi = true
		} else {
			dat_multi = false
		}

		var dat_fmt string
		// Report on flags
		if flags.Tsprec {
            fmt.Printf("16-bit y data\n")
		}
		if flags.Tcgram {
            fmt.Printf("enable fexper\n")
		}
        if flags.Tmulti {
			fmt.Printf("multiple traces\n")
		}
		if flags.Trandm {
            fmt.Printf("arb time (z) values\n")
		}
		if flags.Tordrd {
            fmt.Printf("ordered but uneven subtimes\n")
		}
		if flags.Talabs {
            fmt.Printf("use fcatxt axis not fxtype\n")
		}
		if flags.Txyxys {
            fmt.Printf("each subfile has own x's\n")
			dat_fmt = "-xy"
		} else if flags.Txvals {
            fmt.Printf("floating x-value array preceeds y's\n")
			dat_fmt = "x-y"
		} else {
            fmt.Printf("no x given, must be generated\n")
			dat_fmt = "gx-y"
		}

		x := make([]float64, header.Fnpts)
		sub_pos := 512
		if !flags.Txyxys {
			fmt.Printf("Shit\n")
			if flags.Txvals {
				// Possible issue here as it appears to read in float32 and I'm storing float64... maybe a wait to convert. Future Matt?
				r := bytes.NewReader(content[sub_pos:(sub_pos + 4 * int(header.Fnpts))])
				for i := 0; i < 4*int(header.Fnpts); i++ {
					fmt.Printf("%x\n", content[i])
				}
				if err := binary.Read(r, binary.LittleEndian, &x); err != nil {
					fmt.Println("binary.Read failed:", err)
					os.Exit(3)
				}
				sub_pos = sub_pos + 4 * int(header.Fnpts)
			} else {
				x = Linspace(header.Ffirst, header.Flast, header.Fnpts, true)
			}	
		}

		sub_y := make([][]float64, header.Fnsub)
		for i := range sub_y {
			sub_y[i] = make([]float64, header.Fnpts)
		}

		if dat_fmt == "-xy" and header.Fnpts > 0 {
			for i := 0; i < int(header.Fnsub); i++ {
				fmt.Printf("Need to implement -xy read still\n")
			}
		} else {
			for i := 0; i < int(header.Fnsub); i++ {
				if flags.Txyxys {
					// use points in subfile
					fmt.Printf("Use points in subfile\n")
				} else {
					//use global points
					fmt.Printf("Use global points\n")
				}
			}

		for i := 0; i < int(header.Fnpts); i++ {
			fmt.Printf("%d: %f\n", i, x[i])
		}
		
		// Print everything so it is used at least once
		fmt.Printf("It is a multi file? %t\nData format: %s\n", dat_multi, dat_fmt)

		fmt.Printf("Year: %d\nMonth: %d\nDay: %d\nHour: %d\nMinute: %d\n", date.Year, date.Month, date.Day, date.Hour, date.Minute)

		fmt.Printf("Fexper: %d\nFexp: %d\nFnpts: %d\nFfirst: %f\nFlast: %f\nFnsub: %d\nFxtype: %d\nFytype: %d\nFztype: %d\nFpost: %d\nFdate: %d\nFres: %s\nFsource: %s\nFpeakpt: %d\nFspare: %s\nFcmnt: %s\nFcatxt: %s\nFlogoff: %d\nFmods: %d\nFprocs: %d\nFlevel: %d\nFsampin: %d\nFfactor: %f\nFmethod: %s\nFzinc: %f\nFwplanes: %d\nFwinc: %f\nFwtype: %d\nFreserv: %s\n", header.Fexper, header.Fexp, header.Fnpts, header.Ffirst, header.Flast, header.Fnsub, header.Fxtype, header.Fytype, header.Fztype, header.Fpost, header.Fdate, header.Fres, header.Fsource, header.Fpeakpt, header.Fspare, header.Fcmnt, header.Fcatxt, header.Flogoff, header.Fmods, header.Fprocs, header.Flevel, header.Fsampin, header.Ffactor, header.Fmethod, header.Fzinc, header.Fwplanes, header.Fwinc, header.Fwtype, header.Freserv)

	} else if leader.Fversn == 76 {
		fmt.Printf("SPC file version 76.\n")
	} else if leader.Fversn == 77 {
		fmt.Printf("SPC file version 77.\n")
	} else if leader.Fversn == 207 {
		fmt.Printf("SPC file version 207.\n")
	} else {
		fmt.Printf("SPC file version not implemented yet: %d\n", leader.Fversn)
	}
}

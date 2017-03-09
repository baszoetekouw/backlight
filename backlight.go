package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"bufio"
	"path/filepath"
	"strconv"
	"strings"
	"flag"
	"math"
)

func debug(format string, a...interface{} ) {
	return
	fmt.Printf("DEBUG: ")
	fmt.Printf(format,a...)
}

type Options struct {
	help bool     // show help
	list bool     // show list of all values
	id   int      // select by id
	name string   // select by name
	val  float64  // value to update 
	min  uint     // minimum value
	perc bool     // val is a percentage
	rel  bool     // val is relative
}

func ParseOptions() Options {
	var op_help = flag.Bool(  "h", false, "show this help")
	var op_list = flag.Bool(  "l", false, "show list of all brightness controls")
	var op_id   = flag.Int(   "i",    -1, "select brightness control by id")
	var op_name = flag.String("n",    "", "select brightness control by name")
	var op_val  = flag.String("s",    "", "set/adjust brightness (absolute value, relative value or percentage)")
	var op_min  = flag.Int(   "m",     0, "set minimum brightness (default: 0)")

	flag.Parse()

	// store the results in a nice struct
	var options Options
	options.help = *op_help
	options.list = *op_list
	options.id   = *op_id
	options.name = *op_name
	options.val  = math.NaN()
	options.min  = 0
	options.perc = false
	options.rel  = false


	// now for the manual parsing.
	var str = *op_val
	if len(str)>0 {
		// first check if adjustment is absolute or relative
		if str[0]=='+' || str[0]=='-' {
			options.rel = true
		} 
		// then check if we're dealing with a percentage
		if str[len(str)-1]=='%' {
			options.perc = true
		}
		str = strings.TrimSuffix(str,"%")
		val, err := strconv.ParseFloat(str,64)
		if err!=nil { panic(err) }
		options.val = val

		debug("parsed options; val=%v rel=%v perc=%v\n",options.val,options.rel,options.perc)
	}

	if (*op_min>0) {
		options.min = uint(*op_min)
	}

	// some logic
	if *op_help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if len(*op_val)==0 {
		// no adjustment specified, assume listing
		options.list = true
	}

	if (options.id!=-1 && options.id<0) {
		// illegal brightness icontrol id
		fmt.Printf("Illegal control id (must be positive)\n")
		os.Exit(1)
	}

	if len(*op_val)>0 && !(options.id!=-1 || len(options.name)>0)  {
		// adjustment specified, but no control id.  error out
		fmt.Printf("Brightness control not specified.  Please use -i or -n\n")
		os.Exit(1)
	}

	if options.id!=-1 && len(options.name)>0 {
		// both -n and -i specified
		fmt.Printf("Can't specify backlight control by name and id at the same time\n")
		fmt.Printf("Please use either -i or -n\n")
		os.Exit(1)
	}

	return options
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func FileIsDir(name string) bool {
	if info, err := os.Stat(name); err != nil {
		return false
	} else {
		return info.IsDir()
	}
}

func ScanDir( basedir string ) []string {
	var dirs = []string{}

	entries, err := ioutil.ReadDir(basedir)
	if err!=nil { panic(err) } // TODO: better handling of error

	OUTER:
	for i, e := range entries {
		debug("%d %04o %v %s\n", i, e.Mode(), e.IsDir(), e.Name())
		var d = filepath.Join(basedir,e.Name())
		if !FileIsDir(d) { continue }

		debug("Searching %s\n", d)

		for _, f := range []string{"brightness","max_brightness"} {
			var file = filepath.Join(d,f)
			debug("Checking for %s...", file)
			if !FileExists(file) { 
				debug("no\n")
				continue OUTER
			}
			debug("yes\n")
		}
		// found "brightness" and "max_brightness" in this dir, so save it
		dirs = append(dirs,d)
	}

	return dirs
}

func ScanDirs( dirs...string) []string {
	var result = []string{}
	for _, d := range dirs {
		var thisresult = ScanDir(d)
		result = append(result,thisresult...)
	}
	return result
}

func BLRead(dir string, file string) uint {
	var filename = filepath.Join(dir,file)

	fd, err := os.Open(filename)
	if err!=nil { panic(err) }
	defer fd.Close()

	var scanner = bufio.NewScanner(fd)
	scanner.Split(bufio.ScanWords)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}
	var str = scanner.Text()

	debug("read `%s' from %s\n", str, filename)

	var val = 0
	val, err = strconv.Atoi(str)
	if err!=nil { panic(err) }

	if val<0 {
		// should never occur
		panic("negative brightness")
	}

	return uint(val)

}

func BLReadMax(dir string) uint {
	return BLRead(dir,"max_brightness")
}

func BLReadCurrent(dir string) uint {
	return BLRead(dir,"brightness")
}

func BLWrite(dir string, file string, brightness uint) {
	var filename = filepath.Join(dir,file)

	debug("Writing to file `%v'\n", filename)

	fd, err := os.Create(filename)
	if err!=nil { panic(err) }
	defer fd.Close()

	debug("File opened\n")

	var str = fmt.Sprintf("%v", brightness);
	debug("Writing `%v'\n", str)
	fd.WriteString(str)
}

func BLWriteCurrent(dir string, brightness uint) {
	BLWrite(dir, "brightness", brightness)
}


func SelectDirs(all_dirs []string, options Options) string {

	if options.id!=-1 {
		debug("Selecting by id\n")
		if (options.id < len(all_dirs)) {
			return all_dirs[options.id]
		} else {
			fmt.Printf("Brightness control %v was not found\n", options.id)
			os.Exit(2);
		}
	} else if len(options.name)>0 {
		debug("Selecting by name\n")
		for _, d := range all_dirs {
			if filepath.Base(d) == options.name {
				return d
			}
		}
		fmt.Printf("Brightness control named `%v' not found (try -l)\n", options.name)
		os.Exit(2);
	} else {
		debug("Selecting all\n")
		return ""
	}

	return "NEVER REACHED"
}

func Round(f float64) int {
	if f<0 {
		return -Round(-f)
	} else {
		return int(f+0.5)
	}
}

func CalcBacklight(bl_max uint, bl_min uint, bl_cur uint, val float64, is_rel bool, is_perc bool) uint {
	var bl = 0.0

	// "calculations"
	if is_rel && is_perc {
		bl = float64(bl_cur) + float64(bl_max)*val/100
	} else if is_rel && !is_perc {
		bl = float64(bl_cur) + val
	} else if !is_rel && is_perc {
		bl = val/100*float64(bl_max)
	} else if !is_rel && !is_perc {
		bl = val
	} else {
		panic("Never reached")
	}

	// sanity check 
	if bl<0 || uint(bl)<bl_min {
		return bl_min
	} else if bl>float64(bl_max) {
		return bl_max
	} else {
		return uint(Round(bl))
	}
}


func main() {
	var options = ParseOptions()

	var dirs = ScanDirs("/sys/class/backlight","/sys/class/leds")

	var selected_dir = SelectDirs(dirs, options)
	debug("Selected dir is `%v'\n", selected_dir)

	if options.list {
		debug("Listing all controls\n");
		for i, dir := range dirs {
			if selected_dir!="" && dir!=selected_dir {
				// we only want to list this specific dir
				continue
			}
			var bl_max  = BLReadMax(dir)
			var bl_cur  = BLReadCurrent(dir)
			var bl_perc = 100.0*float64(bl_cur)/float64(bl_max)
			fmt.Printf("%2d  %25s  %4d  %4d  %6.2f\n", i, filepath.Base(dir), bl_cur, bl_max, bl_perc)
		}
		os.Exit(0)
	}

	if !math.IsNaN(options.val) {
		if selected_dir=="" {
			// should never happen
			panic("Invalid state")
		}
		var bl_max = BLReadMax(selected_dir)
		var bl_cur = BLReadCurrent(selected_dir)
		var bl_new = CalcBacklight(bl_max, options.min, bl_cur, options.val, options.rel, options.perc)
		fmt.Printf("Setting backlight to %v\n",bl_new)
		BLWriteCurrent(selected_dir, bl_new)
	}


/*
	dir = Select_dir(dirs)

	bl_max = ReadMax(dir)
	bl_current = ReadCurrent(dir)
	bl_new = CalcBacklight(bl_current, bl_max, options)
	WriteBacklight(dir, bl_new)
*/

}


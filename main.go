package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/scanner"
	"time"
)

var (
	dirFlag     = flag.Bool("d", false, "Print only directories.")
	fileFlag    = flag.Bool("f", false, "Print only non-directories.")
	exeFlag     = flag.Bool("x", false, "Print only if the executable bit is set.")
	rangeFlag   = flag.String("n", "", "Sets the inclusive range for depth filtering.\nThe expected format is \"min,max\" and both are optional.")
	statfmtFlag = flag.String("e", "p", "Specifies the output format.\nThe following characters are accepted:\nU\tOwner name (uid)\nG\tGroup name (gid)\nM\tname of the last user to modify the file\na\tlast access time\nm\tlast modification time\nn\tfinal path element (name)\np\tpath\ns\tsize (bytes)\nx\tpermissions")

	root     int
	command  string
	mindepth int = -1
	maxdepth int = -1
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-dfx] [-n min,max] [-e \"fmt\"] ... [! cmd] \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	if err := parseRange(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
	for n, arg := range args {
		if arg == "!" || strings.HasPrefix(arg, "!") {
			command = strings.Join(args[n+1:], " ")
			args = args[:n]
			break
		}
	}
	for _, arg := range args {
		if strings.HasSuffix(arg, string(os.PathSeparator)) && len(arg) > 1 {
			arg = strings.TrimSuffix(arg, string(os.PathSeparator))
		}
		root = strings.Count(arg, string(os.PathSeparator))
		if err := filepath.Walk(arg, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			var (
				depth int = strings.Count(path, string(os.PathSeparator)) - root
				min       = mindepth
				max       = maxdepth
			)
			if min < 0 {
				min = depth
			}
			if max < 0 {
				max = depth
			}
			if depth < min || depth > max {
				return nil
			}
			if !info.IsDir() && *dirFlag || info.IsDir() && *fileFlag {
				return nil
			}
			return print(path, info)
		}); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}
}

func print(path string, info fs.FileInfo) error {
	var s scanner.Scanner
	s.Init(strings.NewReader(*statfmtFlag))
	s.Mode = scanner.ScanStrings
	s.Whitespace ^= scanner.GoWhitespace
	var d bool
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		d = false
		switch tok {
		case 'U':
			fallthrough
		case 'G':
			fallthrough
		case 'M':
			fallthrough
		case 'a':
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				user, err := user.LookupId(fmt.Sprint(stat.Uid))
				if err == nil {
					switch tok {
					case 'U':
						fmt.Print(user.Uid)
					case 'G':
						fmt.Print(user.Gid)
					case 'M':
						fmt.Print(user.Name)
					case 'a':
						fmt.Print(time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec)))
					}
				}
			}
		case 'm':
			fmt.Printf("%d", info.ModTime().Unix())
		case 'n':
			fmt.Print(info.Name())
		case 's':
			fmt.Printf("%d", info.Size())
		case 'p':
			fmt.Print(path)
		case 'x':
			fmt.Print(info.Mode().Perm().String())
		default:
			fmt.Printf("%c", tok)
			d = true
		}
		if !d {
			fmt.Print(" ")
		}
	}
	if command != "" {
		var s scanner.Scanner
		s.Init(strings.NewReader(command))
		s.Mode = scanner.ScanChars
		s.Whitespace ^= scanner.GoWhitespace
		var sb strings.Builder
		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			if tok == '\\' && s.Peek() == '%' {
				tok = s.Scan()
			}
			sb.WriteRune(tok)
			if tok != '\\' && s.Peek() == '%' {
				tok = s.Scan()
				sb.WriteString(path)
				continue
			}
		}
		output, err := exec.Command("/bin/sh", "-c", sb.String()).Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Print(string(output))
		}
		return nil
	}
	fmt.Print("\n")
	return nil
}

func parseRange() error {
	var (
		err   error
		r     = *rangeFlag
		parts = strings.Split(r, ",")
	)
	if r != "" {
		if len(parts) < 1 {
			maxdepth, err = strconv.Atoi(r)
			if err != nil {
				return err
			}
			return nil
		}
		if len(parts) > 2 {
			return fmt.Errorf("invalid r: %s. expected min,max", r)
		}
		for i, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil {
				if part == "" {
					n = -1
				} else {
					return err
				}
			}
			if i == 0 {
				mindepth = n
			} else {
				maxdepth = n
			}
		}
	}
	return nil
}

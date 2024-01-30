package main

import (
	"errors"
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

	rootDepth int
	minDepth  int
	maxDepth  int
	command   string
)

func print(path string, info fs.FileInfo) error {
	var sb strings.Builder
	var s scanner.Scanner
	if info.Name() == "." || info.Name() == ".." {
		return nil
	}
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
						sb.WriteString(fmt.Sprint(user.Uid))
					case 'G':
						sb.WriteString(fmt.Sprint(user.Gid))
					case 'M':
						sb.WriteString(fmt.Sprint(user.Name))
					case 'a':
						atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
						sb.WriteString(fmt.Sprint(atime))
					}
				}
			}
			// TODO: Handle the niche operating systems (windows, etc..)
		case 'm':
			sb.WriteString(fmt.Sprintf("%d", info.ModTime().Unix()))
		case 'n':
			sb.WriteString(info.Name())
		case 's':
			sb.WriteString(fmt.Sprintf("%d", info.Size()))
		case 'p':
			sb.WriteString(path)
		case 'x':
			sb.WriteString(info.Mode().Perm().String())
		default:
			sb.WriteRune(tok)
			d = true
		}
		if !d {
			sb.WriteByte(' ')
		}
	}
	if command != "" {
		var s scanner.Scanner
		s.Init(strings.NewReader(command))
		s.Mode = scanner.ScanStrings
		s.Whitespace ^= scanner.GoWhitespace
		var sb strings.Builder
		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			sb.WriteRune(tok)
			if tok != '\\' && s.Peek() == '%' {
				tok = s.Scan()
				sb.WriteString(path)
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
	fmt.Println(sb.String())
	return nil
}

func parseRange() error {
	if *rangeFlag != "" {
		parts := strings.Split(*rangeFlag, ",")
		var err error
		if len(parts) > 0 && parts[0] != "" {
			minDepth, err = strconv.Atoi(parts[0])
			if err != nil {
				return errors.New(fmt.Sprintf("invalid min range number: '%s'\n", parts[0]))
			}
		}
		if len(parts) > 1 && parts[1] != "" {
			maxDepth, err = strconv.Atoi(parts[1])
			if err != nil {
				return errors.New(fmt.Sprintf("invalid max range number: '%s'\n", parts[1]))
			}
		}
	}
	return nil
}

func traverse(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	var depth int = strings.Count(path, string(os.PathSeparator))
	if maxDepth > 0 && depth-rootDepth > maxDepth {
		return nil
	}
	if minDepth > 0 && depth-rootDepth < minDepth {
		return nil
	}
	if info.IsDir() && *fileFlag {
		return nil
	}
	if !info.IsDir() && *dirFlag {
		return nil
	}
	if *exeFlag && info.Mode().Perm()&0111 == 0 {
		return nil
	}
	return print(path, info)
}

func main() {
	minDepth = -1
	maxDepth = -1
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-dftxu] [-n min,max] [-e \"fmt\"] ... [! cmd] \n", os.Args[0])
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
		if arg == "!" {
			command = strings.Join(args[n+1:], " ")
			args = args[:n]
			break
		}
	}
	for _, arg := range args {
		if strings.HasSuffix(arg, string(os.PathSeparator)) && len(arg) > 1 {
			arg = strings.TrimSuffix(arg, string(os.PathSeparator))
		}
		rootDepth = strings.Count(arg, string(os.PathSeparator))
		if rootDepth < 1 {
			rootDepth = -1
		}
		err := filepath.Walk(arg, traverse)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}
}

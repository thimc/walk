// Command walk implements the walk command from Plan 9 that walks a directory hierachy.
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
	"time"
)

var (
	isdirectory = flag.Bool("d", false, "Print only directories.")
	isfile      = flag.Bool("f", false, "Print only non-directories.")
	executable  = flag.Bool("x", false, "Print only if the executable bit is set.")
	rangefmt    = flag.String("n", "", "Sets the inclusive range for depth filtering.\nThe expected format is \"min,max\" and both are optional.")
	statfmt     = flag.String("e", "p", "Specifies the output format.\nThe following characters are accepted:\nU\tOwner name (uid)\nG\tGroup name (gid)\nM\tname of the last user to modify the file\na\tlast access time\nm\tlast modification time\nn\tfinal path element (name)\np\tpath\ns\tsize (bytes)\nx\tpermissions")

	cmd      string
	mindepth = -1
	maxdepth = -1
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-dfx] [-n min,max] [-e \"fmt\"] [ name ... ] [! cmd] \n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if err := parseRange(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for n, arg := range args {
		if arg == "!" || strings.HasPrefix(arg, "!") {
			if arg == "!" {
				cmd = strings.Join(args[n+1:], " ")
			} else {
				cmd = strings.Join(args[n:], " ")
			}
			args = args[:n]
			break
		}
	}
	if len(args) < 1 {
		args = []string{"."}
	}
	for _, arg := range args {
		rootdepth := strings.Count(arg, string(os.PathSeparator))
		if err := filepath.Walk(arg, func(path string, fi fs.FileInfo, err error) error {
			if path == "." || path == ".." || !fi.IsDir() && *isdirectory || fi.IsDir() && *isfile {
				return nil
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return nil
			}
			mind, maxd := mindepth, maxdepth
			if maxd < mind {
				maxd = mind
			}
			depth := strings.Count(path, string(os.PathSeparator)) + 1 - rootdepth
			if mind < 0 {
				mind = depth
			}
			if maxd < 0 {
				maxd = depth
			}
			if depth < mind {
				return nil
			}
			if depth > maxd {
				return fs.SkipDir
			}
			if cmd != "" {
				return runCmd(cmd, path)
			}
			return printPath(path, fi)
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func parseRange() error {
	if *rangefmt == "" {
		return nil
	}
	var parts = strings.Split(*rangefmt, ",")
	if len(parts) > 2 {
		return fmt.Errorf("invalid range %s", *rangefmt)
	} else if len(parts) < 2 {
		n, err := strconv.Atoi(*rangefmt)
		if err != nil {
			return err
		}
		maxdepth = n
		mindepth = 0
		return nil
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
	return nil
}

func printPath(path string, fi fs.FileInfo) error {
	for i, r := range *statfmt {
		switch r {
		case 'U', 'G', 'M', 'a':
			stat, ok := fi.Sys().(*syscall.Stat_t)
			if !ok {
				continue
			}
			user, err := user.LookupId(fmt.Sprint(stat.Uid))
			if err != nil {
				continue
			}
			switch r {
			case 'U':
				fmt.Print(user.Uid)
			case 'G':
				fmt.Print(user.Gid)
			case 'M':
				fmt.Print(user.Name)
			case 'a':
				fmt.Print(time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec)).Unix())
			}
		case 'm':
			fmt.Print(fi.ModTime().Unix())
		case 'n':
			fmt.Print(fi.Name())
		case 's':
			fmt.Print(fi.Size())
		case 'p':
			fmt.Print(path)
		case 'x':
			fmt.Print(fi.Mode().Perm().String())
		default:
			fmt.Printf("%c", r)
		}
		if i+1 < len(*statfmt) {
			fmt.Print(" ")
		}
	}
	fmt.Print("\n")
	return nil
}

func runCmd(args, path string) error {
	var sb strings.Builder
	for i, r := range args {
		switch r {
		case '%':
			if i >= 1 && args[i-1] != '\\' {
				sb.WriteString(path)
				continue
			}
			fallthrough
		default:
			sb.WriteRune(r)
		}
	}
	cmd := exec.Command("/bin/sh", "-c", sb.String())
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

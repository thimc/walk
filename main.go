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
	fmt.Fprintf(os.Stderr, "Usage: %s [-dfx] [-n min,max] [-e \"fmt\"] ... [! cmd] \n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		args = []string{"."}
	}
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
	for _, arg := range args {
		if strings.HasSuffix(arg, string(os.PathSeparator)) && len(arg) > 1 {
			arg = strings.TrimSuffix(arg, string(os.PathSeparator))
		}
		rootdepth := strings.Count(arg, string(os.PathSeparator))
		if err := filepath.Walk(arg, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return nil
			}
			if !info.IsDir() && *isdirectory || info.IsDir() && *isfile {
				return nil
			}
			if path == "." || path == ".." {
				return nil
			}
			var (
				depth int = strings.Count(path, string(os.PathSeparator)) - rootdepth
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
			printPath(path, info)
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// parseRange extracts the minimum and maximum directory depth from the [rangefmt] flag.
func parseRange() error {
	var (
		err   error
		r     = *rangefmt
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
			return fmt.Errorf("invalid range %s", r)
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

func printPath(path string, info fs.FileInfo) {
	if cmd != "" {
		runCmd(path)
		return
	}
	for i, r := range *statfmt {
		switch r {
		case 'U', 'G', 'M', 'a':
			stat, ok := info.Sys().(*syscall.Stat_t)
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
			fmt.Print(info.ModTime().Unix())
		case 'n':
			fmt.Print(info.Name())
		case 's':
			fmt.Print(info.Size())
		case 'p':
			fmt.Print(path)
		case 'x':
			fmt.Print(info.Mode().Perm().String())
		default:
			fmt.Printf("%c", r)
		}
		if i+1 < len(*statfmt) {
			fmt.Print(" ")
		}
	}
	fmt.Print("\n")
}

func runCmd(path string) {
	var sb strings.Builder
	for i, r := range cmd {
		switch r {
		case '%':
			if i >= 1 && cmd[i-1] != '\\' {
				sb.WriteString(path)
				continue
			}
			fallthrough
		default:
			sb.WriteRune(r)
		}
	}
	output, err := exec.Command("/bin/sh", "-c", strings.TrimPrefix(sb.String(), "!")).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Print(string(output))
}

// Command camera-shake 跑震屏合成样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/shake"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("camera-shake", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "override", "override / clamp / decay / quota / idem / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "override" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200}, 40, 100)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		s.SetOverride("explosion", shake.Params{Amplitude: 30, DurationMs: 400})
		s.Push("r1", "explosion", 0, 1, 0)
		fmt.Fprintf(stdout, "total=%.0f\n", s.Active(0))
		return 0
	}
	if *sample == "clamp" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200}, 12, 100)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		s.Push("r1", "hit", 0, 1, 0)
		s.Push("r2", "hit", 0, 1, 0)
		fmt.Fprintf(stdout, "total=%.0f\n", s.Active(0))
		return 0
	}
	if *sample == "decay" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200}, 40, 100)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		s.Push("r1", "hit", 0, 1, 0)
		fmt.Fprintf(stdout, "total=%.0f\n", s.Active(100))
		return 0
	}
	if *sample == "quota" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200}, 40, 15)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		s.Push("r1", "hit", 0, 1, 0)
		fmt.Fprintf(stdout, "second=%.0f\n", s.Push("r2", "hit", 0, 1, 0))
		return 0
	}
	if *sample == "idem" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200}, 40, 100)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		s.Push("same", "hit", 0, 1, 0)
		fmt.Fprintf(stdout, "second=%.0f\n", s.Push("same", "hit", 0, 1, 0))
		return 0
	}
	if *sample == "scan" {
		s, err := shake.New(shake.Params{Amplitude: 10, DurationMs: 200000}, 100000000, 100000000)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for index := 0; index < 3000; index++ {
			s.Push(fmt.Sprintf("r-%d", index), "hit", 0, 1, 0)
		}
		for index := 0; index < 3000; index++ {
			s.Active(0)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", s.Scanned())
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample override|clamp|decay|quota|idem|scan")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

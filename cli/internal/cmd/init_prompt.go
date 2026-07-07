package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type initPrompt struct {
	in  io.Reader
	out io.Writer
}

func newInitPrompt(in io.Reader, out io.Writer) *initPrompt {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &initPrompt{in: in, out: out}
}

func isInteractiveTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (p *initPrompt) println(args ...any) {
	fmt.Fprintln(p.out, args...)
}

func (p *initPrompt) printf(format string, args ...any) {
	fmt.Fprintf(p.out, format, args...)
}

func (p *initPrompt) readLine(prompt string) (string, error) {
	if prompt != "" {
		p.printf("%s", prompt)
	}
	reader := bufio.NewReader(p.in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *initPrompt) askYesNo(prompt string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	for {
		line, err := p.readLine(fmt.Sprintf("%s %s: ", prompt, hint))
		if err != nil {
			return defaultYes, err
		}
		if line == "" {
			return defaultYes, nil
		}
		switch strings.ToLower(line) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			p.println("Please enter y or n.")
		}
	}
}

func parseMenuChoice(line string, max int) (int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 1, nil
	}
	n, err := parsePositiveInt(line)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("choose 1-%d", max)
	}
	return n, nil
}

func parsePositiveInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return 0, fmt.Errorf("zero")
	}
	return n, nil
}

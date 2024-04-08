package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

func runConsole() error {
	in := bufio.NewScanner(os.Stdin)

	for {
		os.Stdout.WriteString("> ")
		if !in.Scan() {
			break
		}
		line := in.Text()
		cmdLine := parseCommandLine(line)
		if len(cmdLine) < 1 {
			continue
		}
		if cmdLine[0] == "exit" {
			break
		}
		if err := runCommand(cmdLine); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return nil
}

func parseCommandLine(line string) []string {
	r := strings.NewReader(line)

	var elems []string
	for r.Len() > 0 {
		skipWhitespace(r)
		q, ok := readQuotedString(r)
		if ok {
			elems = append(elems, q)
			continue
		}
		b, ok := readBareWord(r)
		if ok {
			elems = append(elems, b)
		}
	}
	return elems
}

func skipWhitespace(r *strings.Reader) {
	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			return
		}
		if !unicode.IsSpace(c) {
			r.UnreadRune()
			return
		}
	}
}

func readQuotedString(r *strings.Reader) (string, bool) {
	c, _, err := r.ReadRune()
	if err == io.EOF {
		return "", false
	}
	if c != '"' {
		r.UnreadRune()
		return "", false
	}

	var b strings.Builder
	for {
		c, _, err := r.ReadRune()
		if err == io.EOF || c == '"' {
			return b.String(), true
		}
		if c != '\\' {
			b.WriteRune(c)
			continue
		}
		c, _, err = r.ReadRune()
		if err == io.EOF {
			return b.String(), true
		}
		b.WriteRune(c)
	}
}

func readBareWord(r *strings.Reader) (string, bool) {
	var b strings.Builder
	for {
		c, _, err := r.ReadRune()
		// may as well consume the space anyway
		if err == io.EOF || unicode.IsSpace(c) {
			return b.String(), true
		}
		b.WriteRune(c)
	}
}

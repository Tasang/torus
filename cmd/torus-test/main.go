package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Tasang/torus"
)

func main() {
	verbose := flag.Bool("v", false, "verbose output with positions")
	oneline := flag.Bool("1", false, "one token per line")
	mode := flag.String("m", "dict", "tokenization mode: dict, atomic, combined")
	flag.Parse()

	// Parse mode
	var tokMode torus.Mode
	switch strings.ToLower(*mode) {
	case "dict", "d":
		tokMode = torus.ModeDict
	case "atomic", "a":
		tokMode = torus.ModeAtomic
	case "combined", "c":
		tokMode = torus.ModeCombined
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s (use: dict, atomic, combined)\n", *mode)
		os.Exit(1)
	}

	tok := torus.New(torus.WithMode(tokMode))
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			fmt.Println()
			continue
		}

		if *verbose {
			tokens := tok.Tokenize(line)
			for _, t := range tokens {
				fmt.Printf("[%d] %q (%d:%d)\n", t.Position, t.Text, t.Start, t.End)
			}
			if len(tokens) > 0 {
				fmt.Println()
			}
		} else if *oneline {
			tokens := tok.TokenizeToStrings(line)
			for _, t := range tokens {
				fmt.Println(t)
			}
		} else {
			tokens := tok.TokenizeToStrings(line)
			fmt.Println(strings.Join(tokens, "\u200B"))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

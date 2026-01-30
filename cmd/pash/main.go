// Command pash provides a CLI tool for generating password hashes.
// The command reads one password per line on stdin and outputs a hash of that password suitable for use in storage.
// When passwords are entered into a terminal echo is disabled.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/smart-core-os/sc-bos/internal/util/pass"
)

var (
	check string
)

func init() {
	flag.StringVar(&check, "check", "", "Check a password hash instead of generating them")
}

func main() {
	flag.Parse()

	if len(check) > 0 {
		checkHash([]byte(check))
		return
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		readFromStdin()
	} else {
		readFromScanner(bufio.NewScanner(os.Stdin))
	}
}

func checkHash(hash []byte) {
	input, err := readOnePass()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Err reading password: %v\n", err)
		os.Exit(1)
	}

	err = pass.Compare(hash, input)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		_, _ = fmt.Fprintf(os.Stderr, "Read hash: '%v'\n", string(hash))
		os.Exit(2)
	}

	fmt.Println("Success")
}

func readOnePass() ([]byte, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Print("Password: ")
		input, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return input, err
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return nil, scanner.Err()
		}
		return scanner.Bytes(), nil
	}
}

func readFromStdin() {
	for {
		fmt.Print("Password: ")
		input, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Err reading password: %v\n", err)
			os.Exit(1)
		}
		printPassHash(input)
	}
}

func readFromScanner(s *bufio.Scanner) {
	for s.Scan() {
		input := s.Bytes()
		if len(input) == 0 {
			continue // ignore empty lines
		}
		printPassHash(input)
	}
}

func printPassHash(input []byte) {
	hash, err := pass.Hash(input)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Hash error: %v", err)
		return
	}
	printHash(hash)
}

func printHash(hash []byte) {
	fmt.Printf("%v\n", string(hash))
}

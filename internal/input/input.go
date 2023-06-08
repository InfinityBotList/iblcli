// Package input provides secure input methods for iblcli
package input

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func GetPassword(msg string) string {
	fmt.Print(msg + ": ")
	bytepw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		os.Exit(1)
	}
	pass := string(bytepw)

	return pass
}

func GetInput(msg string, check func(s string) bool) string {
	for {
		fmt.Print(msg + ": ")
		var input string
		fmt.Scanln(&input)

		if check(input) {
			return input
		}

		fmt.Println("")
	}
}

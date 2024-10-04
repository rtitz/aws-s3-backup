package generalUtils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// General function to ask for user confirmations
func AskForConfirmation(s string, handleDefault, defaultValue bool) bool {
	reader := bufio.NewReader(os.Stdin)
	answers := "[y/n]"
	if handleDefault && defaultValue {
		answers = "[Y/n]"
	} else if handleDefault && !defaultValue {
		answers = "[y/N]"
	}

	for {
		fmt.Printf("%s %s: ", s, answers)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else if handleDefault && response == "" {
			return defaultValue
		}
	}
}

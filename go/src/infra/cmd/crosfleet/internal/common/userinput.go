// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CLIPrompt prompts the user for a y/n input on CLI.
func CLIPrompt(reason string, defaultResponse bool) (bool, error) {
	if err := prompt(reason, defaultResponse); err != nil {
		return false, err
	}
	for {
		response, err := getPromptResponse()
		if err != nil {
			return false, err
		}
		switch response {
		case "":
			return defaultResponse, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			fmt.Fprintf(os.Stdout, "User aborted session\n")
			return false, nil
		default:
			if err := reprompt(response); err != nil {
				return false, err
			}
		}
	}
}

func prompt(reason string, defaultResponse bool) error {
	b := bufio.NewWriter(os.Stdout)
	fmt.Fprintf(b, "%s\n", reason)
	fmt.Fprintf(b, "Continue?")
	if defaultResponse {
		fmt.Fprintf(b, " [Y/n] ")
	} else {
		fmt.Fprintf(b, " [y/N] ")
	}
	return b.Flush()
}

func getPromptResponse() (string, error) {
	b := bufio.NewReader(os.Stdin)
	i, err := b.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error getting prompt response: %s", err)
	}
	return strings.Trim(strings.ToLower(i), " \n\t"), nil
}

func reprompt(response string) error {
	b := bufio.NewWriter(os.Stdout)
	fmt.Fprintf(b, "\n\tInvalid response %s. Please enter 'y' or 'n': ", response)
	return b.Flush()
}

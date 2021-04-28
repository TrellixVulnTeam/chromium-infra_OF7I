// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// CLIPrompt prompts the user for a y/n input on CLI.
func CLIPrompt(w io.Writer, r io.Reader, reason string, defaultResponse bool) (bool, error) {
	if err := prompt(w, reason, defaultResponse); err != nil {
		return false, err
	}
	for {
		response, err := getPromptResponse(r)
		if err != nil {
			return false, err
		}
		switch response {
		case "":
			return defaultResponse, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			fmt.Fprintf(w, "User aborted session\n")
			return false, nil
		default:
			if err := reprompt(w, response); err != nil {
				return false, err
			}
		}
	}
}

func prompt(w io.Writer, reason string, defaultResponse bool) error {
	b := bufio.NewWriter(w)
	fmt.Fprintf(b, "%s\n", reason)
	fmt.Fprintf(b, "Continue?")
	if defaultResponse {
		fmt.Fprintf(b, " [Y/n] ")
	} else {
		fmt.Fprintf(b, " [y/N] ")
	}
	return b.Flush()
}

func getPromptResponse(r io.Reader) (string, error) {
	b := bufio.NewReader(r)
	i, err := b.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error getting prompt response: %s", err)
	}
	return strings.Trim(strings.ToLower(i), " \n\t"), nil
}

func reprompt(w io.Writer, response string) error {
	b := bufio.NewWriter(w)
	fmt.Fprintf(b, "\n\tInvalid response %s. Please enter 'y' or 'n': ", response)
	return b.Flush()
}

// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	fleet "infra/unifiedfleet/api/v1/proto"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Interactive mode messages for user input
const (
	NameFormat    string = "Name must contain only 4-63 characters, ASCII letters, numbers and special characters -._:"
	InputDetails  string = "Please enter the details: "
	RequiredField string = "is a required field. It cannot be blank/empty."
	WrongInput    string = "\n  WRONG INPUT!!\n"
)

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9-_:.]{4,63}$`)

// Input deatils for the input variable
//
// Key - input variable name
// Desc - description of the variable
// Required - if the variable is a required field
type Input struct {
	Key      string
	Desc     string
	Required bool
}

// GetInteractiveInput collects the scanned string list.
func GetInteractiveInput() []string {
	inputs := make([]string, 0)
	fmt.Print("Please scan: ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		iput := scanner.Text()
		if iput == "" {
			break
		}
		inputs = append(inputs, iput)
		fmt.Print("Continue (please enter without scanning if you finish): ")
	}
	return inputs
}

// GetSwitchInteractiveInput get switch input in interactive mode
func GetSwitchInteractiveInput(s *fleet.Switch) {
	input := &Input{
		Key:      "Name",
		Desc:     NameFormat,
		Required: true,
	}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(InputDetails)
	for input != nil {
		if input.Desc != "" {
			fmt.Println(input.Desc)
		}
		fmt.Print(input.Key, ": ")
		for scanner.Scan() {
			value := scanner.Text()
			if value == "" && input.Required {
				fmt.Println(input.Key, RequiredField)
				fmt.Print(input.Key, ": ")
				continue
			}
			switch input.Key {
			case "Name":
				if !nameRegex.MatchString(value) {
					break
				}
				s.Name = value
				input = &Input{
					Key: "CapacityPort",
				}
			case "CapacityPort":
				if value != "" {
					i, err := strconv.ParseInt(value, 10, 32)
					if err != nil {
						input = &Input{
							Key:  "CapacityPort",
							Desc: fmt.Sprintf("%s", WrongInput),
						}
						break
					}
					s.CapacityPort = int32(i)
				}
				input = nil
			}
			break
		}
	}
}

// ParseJSONFile parses json input from the user provided file.
func ParseJSONFile(jsonFile string, pm proto.Message) error {
	rawText, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return errors.Annotate(err, "parse json file").Err()
	}
	err = jsonpb.Unmarshal(strings.NewReader(string(rawText)), pm)
	return err
}

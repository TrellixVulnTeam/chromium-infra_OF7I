// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"fmt"
	"reflect"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// Local implementation of components.Servod.
type iServod struct {
	dut *tlw.Dut
	a   tlw.Access
}

// DefaultRunner returns runner for current resource name specified per plan.
func (ei *ExecInfo) NewServod() components.Servod {
	return ei.RunArgs.NewServod()
}

// NewServod() returns a struct of type components.Servod that allowes communication with servod service.
func (a *RunArgs) NewServod() components.Servod {
	return &iServod{
		dut: a.DUT,
		a:   a.Access,
	}
}

// Get read value by requested command.
func (s *iServod) Call(ctx context.Context, method string, args ...interface{}) (*xmlrpc.Value, error) {
	res := s.a.CallServod(ctx, &tlw.CallServodRequest{
		Resource: s.dut.Name,
		Method:   method,
		Args:     packToXMLRPCValues(args...),
		Options:  &tlw.ServodOptions{RecoveryMode: true},
	})
	if res.Fault {
		return nil, errors.Reason("call %q", method).Err()
	}
	log.Debug(ctx, "Servod call %q with %v: received %#v", method, args, res.Value.GetScalarOneof())
	return res.Value, nil
}

// Get read value by requested command.
func (s *iServod) Get(ctx context.Context, command string) (*xmlrpc.Value, error) {
	if command == "" {
		return nil, errors.Reason("get: command is empty").Err()
	}
	v, err := s.Call(ctx, "get", command)
	return v, errors.Annotate(err, "get %q", command).Err()
}

// Set sets value to provided command.
func (s *iServod) Set(ctx context.Context, command string, val interface{}) error {
	if command == "" {
		return errors.Reason("set: command is empty").Err()
	}
	if val == nil {
		return errors.Reason("set %q: value is empty", command).Err()
	}
	_, err := s.Call(ctx, "set", command, val)
	return errors.Annotate(err, "set %q with %v", command, val).Err()
}

// Has verifies that command is known.
// Error is returned if the control is not listed in the doc.
func (s *iServod) Has(ctx context.Context, command string) error {
	if command == "" {
		return errors.Reason("has: command not specified").Err()
	}
	if _, err := s.Call(ctx, "doc", command); err == nil {
		return errors.Annotate(err, "has: %q is not know", command).Err()
	}
	return nil
}

// Port provides port used for running servod daemon.
func (s *iServod) Port() int {
	return s.dut.ServoHost.ServodPort
}

// packToXMLRPCValues packs values to XMLRPC structs.
func packToXMLRPCValues(values ...interface{}) []*xmlrpc.Value {
	var r []*xmlrpc.Value
	for _, val := range values {
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case string:
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_String_{
					String_: v,
				},
			})
		case bool:
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_Boolean{
					Boolean: v,
				},
			})
		case int:
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_Int{
					Int: int32(v),
				},
			})
		case float64:
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_Double{
					Double: v,
				},
			})
		default:
			// TODO(otabek@): Extend for more type if required. For now recovery is not using these types.
			message := fmt.Sprintf("%q is not a supported yet to be pack XMLRPC Value ", reflect.TypeOf(val))
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_String_{
					String_: message,
				},
			})
		}
	}
	return r
}

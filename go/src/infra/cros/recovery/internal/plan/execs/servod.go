// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/tlw"
)

// Default servod options.
// For now Repair/Deploy/Verify/Audit/Tests are running with recovery mode as processes always verify servod before provide it for usage.
var defaultServodOptions = &tlw.ServodOptions{RecoveryMode: true}

// ServodCallSet calls servod with set method. Set method used to update the values or call functions with arguments.
func ServodCallSet(ctx context.Context, in *RunArgs, command string, value interface{}) (*tlw.CallServodResponse, error) {
	if command == "" {
		return nil, errors.Reason("servod call set: command is empty").Err()
	}
	if value == nil {
		return nil, errors.Reason("servod call set %q: value is empty", command).Err()
	}
	res := in.Access.CallServod(ctx, &tlw.CallServodRequest{
		Resource: in.DUT.Name,
		Method:   tlw.ServodMethodSet,
		Args:     packToXMLRPCValues(command, value),
		Options:  defaultServodOptions,
	})
	log.Printf("Servod call set %q: received %s", command, res.Value.String())
	if res.Fault {
		return nil, errors.Reason("servod call set %q: received %s", command, res.Value.GetString_()).Err()
	}
	return res, nil
}

// ServodCallGet calls servod with get method. Get method used to read values.
func ServodCallGet(ctx context.Context, in *RunArgs, command string) (*tlw.CallServodResponse, error) {
	if command == "" {
		return nil, errors.Reason("servod call get: command is empty").Err()
	}
	res := in.Access.CallServod(ctx, &tlw.CallServodRequest{
		Resource: in.DUT.Name,
		Method:   tlw.ServodMethodGet,
		Args:     packToXMLRPCValues(command),
		Options:  defaultServodOptions,
	})
	log.Printf("Servod call get %q: received %s", command, res.Value.String())
	if res.Fault {
		return nil, errors.Reason("servod call get %q: received %s", command, res.Value.GetString_()).Err()
	}
	return res, nil
}

// ServodCallHas calls servod with doc method and verify if command is known by servod.
func ServodCallHas(ctx context.Context, in *RunArgs, command string) (*tlw.CallServodResponse, error) {
	if command == "" {
		return nil, errors.Reason("servod call has: command is empty").Err()
	}
	res := in.Access.CallServod(ctx, &tlw.CallServodRequest{
		Resource: in.DUT.Name,
		Method:   tlw.ServodMethodDoc,
		Args:     packToXMLRPCValues(command),
		Options:  defaultServodOptions,
	})
	log.Printf("Servod call has %q: received %s", command, res.Value.String())
	if res.Fault {
		return nil, errors.Reason("servod call has %q: received %s", command, res.Value.GetString_()).Err()
	}
	return res, nil
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
			log.Printf(message)
			r = append(r, &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_String_{
					String_: message,
				},
			})
		}
	}
	return r
}

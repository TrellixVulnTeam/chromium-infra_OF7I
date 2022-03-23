// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package btpeer

import (
	"context"
	"fmt"
	"reflect"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// Call calls XMLRPC server from bluetooth peer.
func Call(ctx context.Context, in tlw.Access, host *tlw.BluetoothPeerHost, method string, args ...interface{}) (*xmlrpc.Value, error) {
	if method == "" {
		return nil, errors.Reason("bluetooth peer call: method name is empty").Err()
	}
	res := in.CallBluetoothPeer(ctx, &tlw.CallBluetoothPeerRequest{
		Resource: host.Name,
		Method:   "GetDetectedStatus",
		Args:     packToXMLRPCValues(args...),
	})
	log.Debugf(ctx, "BluetoothPeer call %q: received %#v", method, res.GetValue())
	if res.GetFault() {
		return nil, errors.Reason("bluetooth peer call %q: received fail flag", method).Err()
	}
	return res.GetValue(), nil
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

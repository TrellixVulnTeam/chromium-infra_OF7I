// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"flag"
	"infra/appengine/crosskylabadmin/site"

	"github.com/maruel/subcommands"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
)

// Constants for possible locations of the CrOSSkylabAdmin service.
const (
	local   = "local"
	staging = "staging"
	prod    = "prod"
)

// CrOSAdminRPCRun contains the common flags for an RPC call to CrOSAdm
//
// The default location for CrOSSkylabAdmin is the staging service.
type crOSAdminRPCRun struct {
	subcommands.CommandRunBase
	local bool
	prod  bool
}

// Register registers common flags for CrOSAdmin RPC commands.
func (r *crOSAdminRPCRun) Register(flags *flag.FlagSet) {
	r.Flags.BoolVar(&r.local, "local", false, "use local service")
	r.Flags.BoolVar(&r.prod, "prod", false, "use production service")
}

// GetType determines the type of CrOSAdmin instance we're supposed to talk to.
func (r *crOSAdminRPCRun) getType() (string, error) {
	switch r.local {
	case true:
		switch r.prod {
		case true:
			return "", errors.Reason("crosadm RPC flags: -local and -prod are mutually exclusive").Err()
		default:
			return local, nil
		}
	default:
		switch r.prod {
		case true:
			return prod, nil
		default:
			return staging, nil
		}
	}
}

// LocalHostPort is the local host and port that CrOSAdmin uses by default.
const localHostPort = "127.0.0.1:8800"

// GetHost gets hostname:port for the correct CrOSAdmin instance.
func (r *crOSAdminRPCRun) GetHost() (string, error) {
	typ, err := r.getType()
	if err != nil {
		return "", err
	}
	switch typ {
	case local:
		return localHostPort, nil
	case prod:
		return site.ProdService, nil
	case staging:
		return site.StagingService, nil
	}
	return "", errors.Reason("internal error").Err()
}

// GetHost gets the PRPC options for the correct CrOSAdmin instance.
func (r *crOSAdminRPCRun) GetOptions() (*prpc.Options, error) {
	typ, err := r.getType()
	if err != nil {
		return nil, err
	}
	options := site.DefaultPRPCOptions
	switch typ {
	case local:
		options.Insecure = true
		return options, nil
	case prod:
		return options, nil
	case staging:
		return options, nil
	}
	return nil, errors.Reason("internal error").Err()
}

// Indent is the indentation to use for JSON. Set it once for consistency.
const indent = "  "

// JsonMarshaler marshals protos to JSON using adminclient-wide settings.
var jsonMarshaler = jsonpb.Marshaler{
	Indent: indent,
}

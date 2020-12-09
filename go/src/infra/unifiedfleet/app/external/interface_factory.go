package external

import (
	"context"
	"net/http"

	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/impl/remote"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/git"
	"infra/libs/sheet"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/util"
)

const defaultCfgService = "luci-config.appspot.com"

var spreadSheetScope = []string{authclient.OAuthScopeEmail, "https://www.googleapis.com/auth/spreadsheets", "https://www.googleapis.com/auth/drive.readonly"}

// InterfaceFactoryKey is the key used to store instance of InterfaceFactory in context.
var InterfaceFactoryKey = util.Key("ufs external-server-interface key")

// CfgInterfaceFactory is a contsructor for a luciconfig.Interface
// For potential unittest usage
type CfgInterfaceFactory func(ctx context.Context) luciconfig.Interface

// MachineDBInterfaceFactory is a constructor for a crimson.CrimsonClient
// For potential unittest usage
type MachineDBInterfaceFactory func(ctx context.Context, host string) (crimson.CrimsonClient, error)

// CrosInventoryInterfaceFactory is a constructor for a invV2Api.InventoryClient
type CrosInventoryInterfaceFactory func(ctx context.Context, host string) (CrosInventoryClient, error)

// SheetInterfaceFactory is a constructor for a sheet.ClientInterface
type SheetInterfaceFactory func(ctx context.Context) (sheet.ClientInterface, error)

// GitInterfaceFactory is a constructor for a git.ClientInterface
type GitInterfaceFactory func(ctx context.Context, gitilesHost, project, branch string) (git.ClientInterface, error)

// InterfaceFactory provides a collection of interfaces to external clients.
type InterfaceFactory struct {
	cfgInterfaceFactory           CfgInterfaceFactory
	machineDBInterfaceFactory     MachineDBInterfaceFactory
	crosInventoryInterfaceFactory CrosInventoryInterfaceFactory
	sheetInterfaceFactory         SheetInterfaceFactory
	gitInterfaceFactory           GitInterfaceFactory
}

// CrosInventoryClient refers to the fake inventory v2 client
type CrosInventoryClient interface {
	ListCrosDevicesLabConfig(ctx context.Context, in *invV2Api.ListCrosDevicesLabConfigRequest, opts ...grpc.CallOption) (*invV2Api.ListCrosDevicesLabConfigResponse, error)
	DeviceConfigsExists(ctx context.Context, in *invV2Api.DeviceConfigsExistsRequest, opts ...grpc.CallOption) (*invV2Api.DeviceConfigsExistsResponse, error)
}

// GetServerInterface retrieves the ExternalServerInterface from context.
func GetServerInterface(ctx context.Context) (*InterfaceFactory, error) {
	if esif := ctx.Value(InterfaceFactoryKey); esif != nil {
		return esif.(*InterfaceFactory), nil
	}
	return nil, errors.Reason("InterfaceFactory not initialized in context").Err()
}

// WithServerInterface adds the external server interface to context.
func WithServerInterface(ctx context.Context) context.Context {
	return context.WithValue(ctx, InterfaceFactoryKey, &InterfaceFactory{
		machineDBInterfaceFactory:     machineDBInterfaceFactoryImpl,
		crosInventoryInterfaceFactory: crosInventoryInterfaceFactoryImpl,
		sheetInterfaceFactory:         sheetInterfaceFactoryImpl,
		gitInterfaceFactory:           gitInterfaceFactoryImpl,
	})
}

// NewMachineDBInterfaceFactory creates a new machine DB interface.
func (es *InterfaceFactory) NewMachineDBInterfaceFactory(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	if es.machineDBInterfaceFactory == nil {
		es.machineDBInterfaceFactory = machineDBInterfaceFactoryImpl
	}
	return es.machineDBInterfaceFactory(ctx, host)
}

func machineDBInterfaceFactoryImpl(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	pclient := &prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	}
	return crimson.NewCrimsonPRPCClient(pclient), nil
}

// NewCrosInventoryInterfaceFactory creates a new CrosInventoryInterface.
func (es *InterfaceFactory) NewCrosInventoryInterfaceFactory(ctx context.Context, host string) (CrosInventoryClient, error) {
	if es.crosInventoryInterfaceFactory == nil {
		es.crosInventoryInterfaceFactory = crosInventoryInterfaceFactoryImpl
	}
	return es.crosInventoryInterfaceFactory(ctx, host)
}

func crosInventoryInterfaceFactoryImpl(ctx context.Context, host string) (CrosInventoryClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return invV2Api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	}), nil
}

// NewSheetInterface creates a new Sheet interface.
func (es *InterfaceFactory) NewSheetInterface(ctx context.Context) (sheet.ClientInterface, error) {
	if es.sheetInterfaceFactory == nil {
		es.sheetInterfaceFactory = sheetInterfaceFactoryImpl
	}
	return es.sheetInterfaceFactory(ctx)
}

func sheetInterfaceFactoryImpl(ctx context.Context) (sheet.ClientInterface, error) {
	// Testing sheet-access@unified-fleet-system-dev.iam.gserviceaccount.com, if works, will move it to config file.
	sheetSA := config.Get(ctx).GetSheetServiceAccount()
	if sheetSA == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "sheet service account is not specified in config")
	}
	t, err := auth.GetRPCTransport(ctx, auth.AsActor, auth.WithServiceAccount(sheetSA), auth.WithScopes(spreadSheetScope...))
	if err != nil {
		return nil, err
	}
	return sheet.NewClient(ctx, &http.Client{Transport: t})
}

// NewGitInterface creates a new git interface.
func (es *InterfaceFactory) NewGitInterface(ctx context.Context, gitilesHost, project, branch string) (git.ClientInterface, error) {
	if es.gitInterfaceFactory == nil {
		es.gitInterfaceFactory = gitInterfaceFactoryImpl
	}
	return es.gitInterfaceFactory(ctx, gitilesHost, project, branch)
}

func gitInterfaceFactoryImpl(ctx context.Context, gitilesHost, project, branch string) (git.ClientInterface, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesapi.OAuthScope))
	if err != nil {
		return nil, err
	}
	return git.NewClient(ctx, &http.Client{Transport: t}, "", gitilesHost, project, branch)
}

// NewCfgInterface creates a new config interface.
func (es *InterfaceFactory) NewCfgInterface(ctx context.Context) luciconfig.Interface {
	if es.cfgInterfaceFactory == nil {
		es.cfgInterfaceFactory = cfgInterfaceFactoryImpl
	}
	return es.cfgInterfaceFactory(ctx)
}

func cfgInterfaceFactoryImpl(ctx context.Context) luciconfig.Interface {
	cfgService := config.Get(ctx).LuciConfigService
	if cfgService == "" {
		cfgService = defaultCfgService
	}
	return remote.New(cfgService, false, func(ctx context.Context) (*http.Client, error) {
		t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
		if err != nil {
			return nil, err
		}
		return &http.Client{Transport: t}, nil
	})
}

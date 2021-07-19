module infra

go 1.16

require (
	cloud.google.com/go v0.87.0
	cloud.google.com/go/bigquery v1.19.0
	cloud.google.com/go/datastore v1.5.0
	cloud.google.com/go/firestore v1.5.0
	cloud.google.com/go/pubsub v1.12.2
	cloud.google.com/go/storage v1.16.0
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/StackExchange/wmi v1.2.0
	github.com/VividCortex/godaemon v1.0.0
	github.com/aclements/go-moremath v0.0.0-20210112150236-f10218a38794
	github.com/andygrunwald/go-gerrit v0.0.0-20210709065208-9d38b0be0268
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210707120613-a80295a60b3e // indirect
	github.com/aws/aws-sdk-go v1.40.0 // indirect
	github.com/bazelbuild/remote-apis-sdks v0.0.0-20210719032003-1cec173a5bf7
	github.com/bmatcuk/doublestar v1.3.4
	github.com/containerd/containerd v1.5.3 // indirect
	github.com/danjacques/gofslock v0.0.0-20200623023034-5d0bd0fa6ef0
	github.com/docker/docker v20.10.7+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.6.0
	github.com/google/go-cmp v0.5.6
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/subcommands v1.2.0
	github.com/google/tink/go v1.6.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/googleapis/google-cloud-go-testing v0.0.0-20210427215956-1487aa9ec5b0
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/julienschmidt/httprouter v1.3.0
	github.com/kr/pretty v0.2.1
	github.com/kylelemons/godebug v1.1.0
	github.com/maruel/subcommands v1.1.0
	github.com/mattes/migrate v3.0.1+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/otiai10/copy v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/sergi/go-diff v1.2.0
	github.com/shirou/gopsutil v3.21.6+incompatible
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/smartystreets/goconvey v1.6.4
	github.com/tklauser/go-sysconf v0.3.7 // indirect
	github.com/waigani/diffparser v0.0.0-20190828052634-7391f219313d
	go.chromium.org/chromiumos/config/go v0.0.0-20210225201405-02ec5b5e84b7
	go.chromium.org/chromiumos/infra/proto/go v0.0.0-00010101000000-000000000000
	go.chromium.org/luci v0.0.0-20201029184154-594d11850ebf
	go.opencensus.io v0.23.0
	go.skia.org/infra v0.0.0-20210713223222-9189a6f07020
	golang.org/x/build v0.0.0-20210714161125-11654a71d3e2
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	golang.org/x/time v0.0.0-20210611083556-38a9dc6acbc6
	golang.org/x/tools v0.1.5
	gonum.org/v1/gonum v0.9.3
	google.golang.org/api v0.50.0
	google.golang.org/appengine v1.6.7
	google.golang.org/genproto v0.0.0-20210714021259-044028024a4f
	google.golang.org/grpc v1.39.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	howett.net/plist v0.0.0-20201203080718-1454fab16a06
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/utils v0.0.0-20210709001253-0e1f9d693477 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

// See https://github.com/google/cel-go/issues/441.
exclude github.com/antlr/antlr4 v0.0.0-20200503195918-621b933c7a7f

// Versions >=v0.7.0 break infra/cros/cmd/tclint.
replace github.com/google/cel-go => github.com/google/cel-go v0.6.0

// k8s.io/klog/v2 needs this specific version and fails to compile otherwise.
replace github.com/go-logr/logr => github.com/go-logr/logr v0.4.0

// Infra modules are included via gclient DEPS.
replace (
	go.chromium.org/chromiumos/config/go => ../go.chromium.org/chromiumos/config/go
	go.chromium.org/chromiumos/infra/proto/go => ../go.chromium.org/chromiumos/infra/proto/go
	go.chromium.org/luci => ../go.chromium.org/luci
)

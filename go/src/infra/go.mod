module infra

go 1.16

require (
	cloud.google.com/go v0.100.2
	cloud.google.com/go/appengine v1.2.0
	cloud.google.com/go/bigquery v1.28.0
	cloud.google.com/go/cloudtasks v0.1.0
	cloud.google.com/go/compute v1.3.0
	cloud.google.com/go/datastore v1.5.0
	cloud.google.com/go/firestore v1.5.0
	cloud.google.com/go/iam v0.2.0 // indirect
	cloud.google.com/go/pubsub v1.17.0
	cloud.google.com/go/spanner v1.29.0
	cloud.google.com/go/storage v1.21.0
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/StackExchange/wmi v1.2.1
	github.com/VividCortex/godaemon v1.0.0
	github.com/aclements/go-moremath v0.0.0-20210112150236-f10218a38794
	github.com/alecthomas/participle/v2 v2.0.0-alpha7
	github.com/andygrunwald/go-gerrit v0.0.0-20210726065827-cc4e14e40b5b
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210907221601-4f80a5e09cd0 // indirect
	github.com/bazelbuild/remote-apis-sdks v0.0.0-20220422144733-8780f11b1bb2
	github.com/bmatcuk/doublestar v1.3.4
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/danjacques/gofslock v0.0.0-20220131014315-6e321f4509c8
	github.com/docker/docker v20.10.8+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/go-logr/logr v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.7.3
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.6.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/subcommands v1.2.0
	github.com/google/uuid v1.3.0
	github.com/googleapis/gax-go/v2 v2.1.1
	github.com/googleapis/google-cloud-go-testing v0.0.0-20210719221736-1c9a4c676720
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jdxcode/netrc v0.0.0-20210204082910-926c7f70242a
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julienschmidt/httprouter v1.3.0
	github.com/klauspost/compress v1.13.5
	github.com/kr/pretty v0.3.0
	github.com/kylelemons/godebug v1.1.0
	github.com/linkedin/goavro/v2 v2.11.0
	github.com/maruel/subcommands v1.1.1
	github.com/mattes/migrate v3.0.1+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/opencontainers/image-spec v1.0.1
	github.com/otiai10/copy v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.6.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/smartystreets/goconvey v1.7.2
	github.com/waigani/diffparser v0.0.0-20190828052634-7391f219313d
	go.chromium.org/chromiumos/config/go v0.0.0-20211012171127-50826c369fec
	go.chromium.org/chromiumos/infra/proto/go v0.0.0-00010101000000-000000000000
	go.chromium.org/luci v0.0.0-20201029184154-594d11850ebf
	go.opencensus.io v0.23.0
	go.skia.org/infra v0.0.0-20210913170701-f020cec45197
	golang.org/x/build v0.0.0-20210913192547-14e3e09d6b10
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/mobile v0.0.0-20191031020345-0945064e013a
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	golang.org/x/tools v0.1.8-0.20211014194737-fc98fb2abd48
	golang.org/x/tools/gopls v0.7.3
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gonum.org/v1/gonum v0.9.3
	google.golang.org/api v0.70.0
	google.golang.org/appengine v1.6.7
	google.golang.org/appengine/v2 v2.0.1
	google.golang.org/genproto v0.0.0-20220222213610-43724f9ea8cf
	google.golang.org/grpc v1.40.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/klog/v2 v2.20.0 // indirect
	k8s.io/metrics v0.22.1
	k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e // indirect
	sigs.k8s.io/yaml v1.3.0
)

// See https://github.com/google/cel-go/issues/441.
exclude github.com/antlr/antlr4 v0.0.0-20200503195918-621b933c7a7f

// The following requirements were added on 2022-02-22 to prevent blocking deployment of
// go111 applications.
exclude google.golang.org/grpc v1.44.0

// The next version uses errors.Is(...) and no longer works on GAE go111.
replace golang.org/x/net => golang.org/x/net v0.0.0-20210503060351-7fd8e65b6420

// More recent versions break sysmon tests, crbug.com/1142700.
replace github.com/shirou/gopsutil => github.com/shirou/gopsutil v2.20.10-0.20201018091616-3202231bcdbd+incompatible

// Apparently checking out NDKs at head isn't really safe.
replace golang.org/x/mobile => golang.org/x/mobile v0.0.0-20170111200746-6f0c9f6df9bb

// Version 1.2.0 has a bug: https://github.com/sergi/go-diff/issues/115
exclude github.com/sergi/go-diff v1.2.0

// Infra modules are included via gclient DEPS.
replace (
	go.chromium.org/chromiumos/config/go => ../go.chromium.org/chromiumos/config/go
	go.chromium.org/chromiumos/infra/proto/go => ../go.chromium.org/chromiumos/infra/proto/go
	go.chromium.org/luci => ../go.chromium.org/luci
)

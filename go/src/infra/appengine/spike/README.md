# Spike

Spike is a cloud service which can collect, process and save build logs and use
those logs to prepare an initial build manifest. The build logs here refer to
those that are necessary to generate
[provenance](https://chrome-internal.googlesource.com/infra/infra_internal/+/44b7475364615f237ec8fa7fd64d2f0b44fdfe59/go/src/infra_internal/tools/provenance/proto/provenance.proto#135)
for software artifacts built from LUCI infrastructure.

Tracking bug: [crbug.com/1276644](https://crbug.com/1276644)

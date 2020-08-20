package config

import (
	"context"
	"testing"

	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	"go.chromium.org/luci/config/impl/memory"
	gae "go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/caching"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	Convey("With mocks", t, func() {
		configs := map[config.Set]memory.Files{
			"services/${appid}": map[string]string{},
		}
		mockConfig := func(body string) {
			configs["services/${appid}"][cachedCfg.Path] = body
		}

		ctx := gae.Use(context.Background())
		ctx = cfgclient.Use(ctx, memory.New(configs))
		ctx = caching.WithEmptyProcessCache(ctx)

		Convey("No config", func() {
			So(Set(ctx), ShouldErrLike, "no such config")

			cfg, err := Get(ctx)
			So(cfg, ShouldBeNil)
			So(err, ShouldErrLike, "failed to fetch cached config")
		})

		Convey("Broken config", func() {
			mockConfig("broken")
			So(Set(ctx), ShouldErrLike, "validation errors")
		})

		Convey("Good config", func() {
			mockConfig(`
        hosts {
          name: "chromium"
          repos {
            name: "chromium/src"
            priority: true
          }
          repos {
            name: "chromium/src/codesearch"
            do_not_index: true
          }
        }
        hosts {
          name: "webrtc"
          repos {
            name: "src"
            refs: "refs/heads/.*"
            refs: "refs/branch-heads/.*"
            exclude_refs: "refs/branch-heads/7.*"
          }
        }
			`)
			So(Set(ctx), ShouldBeNil)

			cfg, err := Get(ctx)
			So(err, ShouldBeNil)
			So(cfg, ShouldResembleProto, &Config{
				Hosts: []*Host{
					{
						Name: "chromium",
						Repos: []*Repository{
							{
								Name: "chromium/src",
								Indexing: &Repository_Priority{
									Priority: true,
								},
							},
							{
								Name: "chromium/src/codesearch",
								Indexing: &Repository_DoNotIndex{
									DoNotIndex: true,
								},
							},
						},
					},
					{
						Name: "webrtc",
						Repos: []*Repository{
							{
								Name: "src",
								Refs: []string{
									"refs/heads/.*",
									"refs/branch-heads/.*",
								},
								ExcludeRefs: []string{"refs/branch-heads/7.*"},
							},
						},
					},
				},
			})
		})
	})
}

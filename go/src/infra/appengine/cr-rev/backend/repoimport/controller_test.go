package repoimport

import (
	"context"
	"errors"
	"infra/appengine/cr-rev/common"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func factoryFunc(m map[common.GitRepository]Importer) ImporterFactory {
	return func(ctx context.Context, repo common.GitRepository) Importer {
		return m[repo]
	}
}

func TestController(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("No errors", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		Convey("one repo", func() {
			repo := common.GitRepository{}
			mock := NewMockImporter(mockCtrl)
			mock.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				cancel()
				return nil
			}).Times(1)

			c := NewController(factoryFunc(map[common.GitRepository]Importer{
				repo: mock,
			}))
			c.Index(repo)
			c.Start(ctx)
		})

		Convey("two repos", func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			repo1 := common.GitRepository{Name: "foo"}
			repo2 := common.GitRepository{Name: "bar"}
			mock1 := NewMockImporter(mockCtrl)
			mock1.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				return nil
			}).Times(1)
			mock2 := NewMockImporter(mockCtrl)
			mock2.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				cancel()
				return nil
			}).Times(1)
			c := NewController(factoryFunc(map[common.GitRepository]Importer{
				repo1: mock1,
				repo2: mock2,
			}))
			c.Index(repo1)
			c.Index(repo2)
			c.Start(ctx)
		})
	})

	Convey("With errors", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		Convey("one repo", func() {
			repo := common.GitRepository{}
			mock := NewMockImporter(mockCtrl)
			mock.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				cancel()
				return errors.New("Error")
			}).Times(1)
			c := NewController(factoryFunc(map[common.GitRepository]Importer{
				repo: mock,
			}))
			c.Index(repo)
			c.Start(ctx)
		})

		Convey("two repos", func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			repo1 := common.GitRepository{Name: "foo"}
			repo2 := common.GitRepository{Name: "bar"}
			mock1 := NewMockImporter(mockCtrl)
			mock1.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				return errors.New("error foo")
			}).Times(1)
			mock2 := NewMockImporter(mockCtrl)
			mock2.EXPECT().Run(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				cancel()
				return nil
			}).Times(1)
			c := NewController(factoryFunc(map[common.GitRepository]Importer{
				repo1: mock1,
				repo2: mock2,
			}))
			c.Index(repo1)
			c.Index(repo2)
			c.Start(ctx)
		})
	})
}

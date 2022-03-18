package basic

import (
	"sync"
	"context"

	"github.com/paul-nelson-baker/go-testainer"
	"github.com/paul-nelson-baker/go-testainer/internal/common"
)

var (
	globalClientOnce sync.Once
	globalClient     testainer.Testainer[testainer.Config, testainer.ContainerDetails] = nil
)

func NewBasic() (testainer.Testainer[testainer.Config, testainer.ContainerDetails], error) {
	return testainer.New[testainer.Config, testainer.ContainerDetails](common.Identity[testainer.Config], common.Identity[testainer.ContainerDetails])
}

func Use(ctx context.Context, config testainer.Config, callback testainer.CallbackFunc[testainer.ContainerDetails]) error {
	globalClientInit()
	return globalClient.Use(ctx, config, callback)
}

func Run(ctx context.Context, config testainer.Config) (*testainer.ContainerDetails, testainer.CleanupFunc, error) {
	globalClientInit()
	return globalClient.Run(ctx, config)
}

func globalClientInit() {
	globalClientOnce.Do(func() {
		var err error
		globalClient, err = NewBasic()
		if err != nil {
			panic("could not initialize testainer client: " + err.Error())
		}
	})
}

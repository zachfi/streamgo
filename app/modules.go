package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/pkg/errors"

	"github.com/zachfi/streamgo/modules/ripper"
)

const (
	Server string = "server"

	Ripper string = "ripper"

	All string = "all"
)

func (a *App) setupModuleManager() error {
	mm := modules.NewManager(kitlog.NewLogfmtLogger(os.Stderr))
	mm.RegisterModule(Server, a.initServer, modules.UserInvisibleModule)

	mm.RegisterModule(Ripper, a.initRipper)

	mm.RegisterModule(All, nil)

	deps := map[string][]string{
		// Server:       nil,
		Ripper: {Server},

		All: {Ripper},
	}

	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}

	a.ModuleManager = mm

	return nil
}

func (a *App) initRipper() (services.Service, error) {
	r, err := ripper.New(a.cfg.Ripper, a.logger)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init "+metricsNamespace)
	}

	return r, nil
}

func (a *App) initServer() (services.Service, error) {
	a.cfg.Server.MetricsNamespace = metricsNamespace
	a.cfg.Server.ExcludeRequestInLog = true
	a.cfg.Server.RegisterInstrumentation = true
	a.cfg.Server.Log = kitlog.NewLogfmtLogger(os.Stderr)

	server, err := server.New(a.cfg.Server)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create server")
	}

	servicesToWaitFor := func() []services.Service {
		svs := []services.Service(nil)
		for m, s := range a.serviceMap {
			// Server should not wait for itself.
			if m != Server {
				svs = append(svs, s)
			}
		}

		return svs
	}

	a.Server = server

	serverDone := make(chan error, 1)

	runFn := func(ctx context.Context) error {
		go func() {
			defer close(serverDone)
			serverDone <- server.Run()
		}()

		select {
		case <-ctx.Done():
			return nil
		case err := <-serverDone:
			if err != nil {
				return err
			}

			return fmt.Errorf("server stopped unexpectedly")
		}
	}

	stoppingFn := func(_ error) error {
		// wait until all modules are done, and then shutdown server.
		for _, s := range servicesToWaitFor() {
			_ = s.AwaitTerminated(context.Background())
		}

		// shutdown HTTP and gRPC servers (this also unblocks Run)
		server.Shutdown()

		// if not closed yet, wait until server stops.
		<-serverDone
		slog.Info("server stopped")
		return nil
	}

	return services.NewBasicService(nil, runFn, stoppingFn), nil
}

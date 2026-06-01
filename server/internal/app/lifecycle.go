package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

func (a *Application) Run(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", a.Config.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		a.Log.Info("http server started", "addr", listener.Addr().String())
		err := a.HTTPServer.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("http server: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		return a.shutdown(ctx)
	case err := <-serverErr:
		return err
	}
}

func (a *Application) shutdown(ctx context.Context) error {
	a.Log.Info("application stopping")

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.Config.ShutdownTimeout)
	defer cancel()

	var errs []error
	if a.HTTPServer != nil {
		if err := a.HTTPServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown http: %w", err))
		}
	}

	if a.MongoClient != nil {
		if err := a.MongoClient.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("disconnect mongodb: %w", err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	a.Log.Info("application stopped")
	return nil
}

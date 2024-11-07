package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)
func (a *applicationDependencies) serve() error {
    apiServer := &http.Server{
        Addr:         fmt.Sprintf(":%d", a.config.port),  // Use a.config instead of a.settings
        Handler:      a.routes(),
        IdleTimeout:  time.Minute,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        ErrorLog:     slog.NewLogLogger(a.logger.Handler(), slog.LevelError),
    }

    // Create a channel to track errors during the shutdown process
    shutdownError := make(chan error)

    // Goroutine to listen for shutdown signals
    go func() {
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        s := <-quit

        // Log the shutdown signal
        a.logger.Info("shutting down server", "signal", s.String())

        // Create a context with a 30-second timeout for graceful shutdown
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Send the shutdown error if it occurs
        shutdownError <- apiServer.Shutdown(ctx)
    }()

    // Start the server
    a.logger.Info("starting server", "address", apiServer.Addr, "environment", a.config.environment)  // Use a.config instead of a.settings
    err := apiServer.ListenAndServe()
    if !errors.Is(err, http.ErrServerClosed) {
        return err
    }

    // Wait for any shutdown errors
    err = <-shutdownError
    if err != nil {
        return err
    }

    // Log that the server stopped successfully
    a.logger.Info("stopped server", "address", apiServer.Addr)

    return nil
}

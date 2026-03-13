package main

import (
	"context"

	"keywordAutomator/backend"
)

// App is the Wails application struct.
// It holds the context and the backend.
type App struct {
	ctx     context.Context
	Backend *backend.Backend
}

func NewApp() *App {
	return &App{
		Backend: backend.New(),
	}
}

// setupBackend is called by Wails after the window is created.
// This is the single place where all services are initialised.
func (a *App) setupBackend(ctx context.Context) {
	a.ctx = ctx
	if err := a.Backend.Setup(ctx); err != nil {
		// The app will function but AI features will not work.
		// The frontend should handle missing-API-key state gracefully.
	}
}

// shutdownBackend is called by Wails just before the app exits.
func (a *App) shutdownBackend(ctx context.Context) {
	a.Backend.Shutdown()
}

package backend

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"keywordAutomator/backend/commands/keywords"
	"keywordAutomator/backend/commands/settings"
	"keywordAutomator/backend/commands/upload"
	"keywordAutomator/internal/ai"
	"keywordAutomator/internal/config"
	"keywordAutomator/internal/jobs"

	"github.com/joho/godotenv"
)

type Backend struct {
	Settings *settings.SettingsCommand
	Keywords *keywords.KeywordsCommand
	Upload   *upload.UploadCommand
}

func New() *Backend {
	return &Backend{
		Settings: &settings.SettingsCommand{},
		Keywords: &keywords.KeywordsCommand{},
		Upload:   &upload.UploadCommand{},
	}
}

func (b *Backend) Setup(ctx context.Context) error {
	initLogger()
	loadDotEnv()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return err
	}

	b.Upload.Setup(ctx)

	jobMgr := jobs.NewManager()
	jobMgr.StartCleanup(10*time.Minute, 30*time.Minute)

	// reinitAI rebuilds the AI client and rewires Keywords with the new client.
	// Captured by Settings.Setup so it can be called when the user saves a new key.
	reinitAI := func(apiKey string) error {
		client, err := ai.NewClient(ctx, apiKey)
		if err != nil {
			return err
		}
		b.Keywords.Setup(ctx, jobMgr, client, client)
		return nil
	}

	b.Settings.Setup(cfg, reinitAI)

	// Try to init AI now. If the key is missing, the frontend will redirect to
	// Settings — the user can enter a key there and reinitAI will be called then.
	if cfg.GeminiAPIKey != "" {
		if err := reinitAI(cfg.GeminiAPIKey); err != nil {
			slog.Error("Failed to initialise AI client", "error", err)
		}
	}

	slog.Info("Backend ready")
	return nil
}

func (b *Backend) Shutdown() {
	slog.Info("Backend shutting down")
}

func initLogger() {
	file, _ := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger := slog.New(slog.NewTextHandler(multiWriter, nil))
	slog.SetDefault(logger)
}

func loadDotEnv() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found")
	}
}

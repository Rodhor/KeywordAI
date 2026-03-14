package backend

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"keywordAutomator/backend/commands/keywords"
	"keywordAutomator/backend/commands/upload"
	"keywordAutomator/internal/ai"
	"keywordAutomator/internal/jobs"

	"github.com/joho/godotenv"
)

type Backend struct {
	Keywords *keywords.KeywordsCommand
	Upload   *upload.UploadCommand
}

func New() *Backend {
	return &Backend{
		Keywords: &keywords.KeywordsCommand{},
		Upload:   &upload.UploadCommand{},
	}
}

func (b *Backend) Setup(ctx context.Context) error {
	initLogger()
	loadDotEnv()

	geminiClient, err := ai.NewClient(ctx)
	if err != nil {
		slog.Error("Failed to initialise AI client", "error", err)
		return err
	}

	jobMgr := jobs.NewManager()
	jobMgr.StartCleanup(10*time.Minute, 30*time.Minute)

	b.Keywords.Setup(ctx, jobMgr, geminiClient, geminiClient)
	b.Upload.Setup(ctx)

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

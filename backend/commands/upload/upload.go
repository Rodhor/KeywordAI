package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UploadCommand struct {
	ctx context.Context
}

func (u *UploadCommand) Setup(ctx context.Context) {
	u.ctx = ctx
}

// PickAndSaveCSV opens the native OS file dialog, then copies the selected
// file into tmp/ and returns the filename. Returns an empty string if the
// user cancels.
func (u *UploadCommand) PickAndSaveCSV() (string, error) {
	path, err := runtime.OpenFileDialog(u.ctx, runtime.OpenDialogOptions{
		Title: "Select keyword CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}
	return u.saveCSV(path)
}

func (u *UploadCommand) saveCSV(sourcePath string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(sourcePath), ".csv") {
		return "", fmt.Errorf("only .csv files are supported")
	}

	if err := os.MkdirAll("tmp", 0755); err != nil {
		return "", fmt.Errorf("could not create tmp directory: %w", err)
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %w", err)
	}
	defer src.Close()

	filename := filepath.Base(sourcePath)
	destPath := filepath.Join("tmp", filename)

	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("could not create destination file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("could not copy file: %w", err)
	}

	return filename, nil
}

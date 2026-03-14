package settings

import "keywordAutomator/internal/config"

type Settings struct {
	GeminiAPIKey     string
	GoogleAuthAPIKey string
}

type SettingsCommand struct {
	cfg      *config.Config
	reinitAI func(ApiKey string) error
}

func (s *SettingsCommand) Setup(cfg *config.Config, reinitFn func(string) error) {
	s.cfg = cfg
	s.reinitAI = reinitFn
}

func (s *SettingsCommand) GetSettings() (*Settings, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	return &Settings{
		GeminiAPIKey:     cfg.GeminiAPIKey,
		GoogleAuthAPIKey: cfg.GoogleAuthAPIKey,
	}, nil
}

func (s *SettingsCommand) HasRequiredConfig() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return cfg.GeminiAPIKey != ""
}

func (s *SettingsCommand) Save(newSettings Settings) error {
	s.cfg.GeminiAPIKey = newSettings.GeminiAPIKey
	s.cfg.GoogleAuthAPIKey = newSettings.GoogleAuthAPIKey

	if err := s.cfg.Save(); err != nil { // save first
		return err
	}

	return s.reinitAI(newSettings.GeminiAPIKey) // reinit after
}

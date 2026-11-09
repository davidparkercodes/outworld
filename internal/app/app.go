package app

import (
	"fmt"
	"os"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/outworld"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

const DefaultConfigPath = "outworld.yaml"

func ConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("OUTWORLD_CONFIG"); env != "" {
		return env
	}
	return DefaultConfigPath
}

func BuildEngine(cfg *config.Config) (*outworld.Engine, error) {
	key, err := vault.LoadKey(cfg.Vault.KeyEnv, cfg.Resolve(cfg.Vault.KeyFile))
	if err != nil {
		return nil, err
	}
	v, err := vault.OpenFile(cfg.Resolve(cfg.Vault.Path), key)
	if err != nil {
		return nil, err
	}
	pol, err := cfg.EnginePolicy()
	if err != nil {
		return nil, err
	}
	opts := outworld.Options{Vault: v, Policy: pol}
	if cfg.Policy.Dictionary != "" {
		d, err := detect.LoadDictionary(cfg.Resolve(cfg.Policy.Dictionary))
		if err != nil {
			return nil, err
		}
		opts.Dictionary = d
	}
	if cfg.NER.URL != "" {
		n := detect.NewNERDetector(cfg.NER.URL, time.Duration(cfg.NER.Timeout))
		if cfg.NER.Threshold > 0 {
			n.Threshold = cfg.NER.Threshold
		}
		if len(cfg.NER.Labels) > 0 {
			n.Labels = map[string]entity.Type{}
			for label, typ := range cfg.NER.Labels {
				t, err := entity.ParseType(typ)
				if err != nil {
					return nil, fmt.Errorf("ner.labels.%s: %w", label, err)
				}
				n.Labels[label] = t
			}
		}
		opts.NER = n
		opts.NEROptional = cfg.NER.Optional
	}
	return outworld.New(opts)
}

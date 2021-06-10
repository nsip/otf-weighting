package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config :
type Config struct {
	FatalOnErr       bool
	InboundType      string
	InboundMustArray bool
	InStorage        string
	OutboundType     string
	OutStorage       string
	Service          struct {
		Port int
		API  string
	}
	Weighting struct {
		ReferPrevRecord bool
		StudentIDPath   string
		DomainPath      string
		TimePath        string
		ScorePath       string
	}
}

// GetConfig :
func GetConfig(configs ...string) *Config {
	for _, config := range configs {
		cfg := &Config{}
		_, err := toml.DecodeFile(config, cfg)
		if err != nil {
			continue
		}

		// Directory Process
		cfg.InStorage = filepath.Clean(cfg.InStorage)
		cfg.OutStorage = filepath.Clean(cfg.OutStorage)

		// API Process
		cfg.Service.API = withSlash(cfg.Service.API)

		// Env
		os.Setenv("FatalOnErr", strconv.FormatBool(cfg.FatalOnErr))

		return cfg
	}
	log.Fatalln("Report Config File is Missing or Error")
	return nil
}

func withSlash(str string) string {
	return "/" + strings.Trim(str, "/")
}

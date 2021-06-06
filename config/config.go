package config

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config :
type Config struct {
	InboundFileType  string
	InboundMustArray bool
	AuditDir         string
	Service          struct {
		Port int
		API  string
	}
	Weighting struct {
		StudentIDPath string
		DomainPath    string
		ScorePath     string
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
		cfg.AuditDir = filepath.Clean(cfg.AuditDir)

		// API Process
		cfg.Service.API = withSlash(cfg.Service.API)

		return cfg
	}
	log.Fatalln("Report Config File is Missing or Error")
	return nil
}

func withSlash(str string) string {
	return "/" + strings.Trim(str, "/")
}

package source

import (
	"fmt"
)

const (
	ConfigKeyEnvironment   = "environment"
	ConfigKeyClientId      = "clientId"
	ConfigKeyClientSecret  = "clientSecret"
	ConfigKeyUsername      = "username"
	ConfigKeyPassword      = "password"
	ConfigKeySecurityToken = "securityToken"
	ConfigKeyPushTopicName = "pushTopicName"
)

type Config struct {
	Environment   string
	ClientId      string
	ClientSecret  string
	Username      string
	Password      string
	SecurityToken string
	PushTopicName string
}

func ParseConfig(cfgRaw map[string]string) (Config, error) {
	cfg := Config{
		Environment:   cfgRaw[ConfigKeyEnvironment],
		ClientId:      cfgRaw[ConfigKeyClientId],
		ClientSecret:  cfgRaw[ConfigKeyClientSecret],
		Username:      cfgRaw[ConfigKeyUsername],
		Password:      cfgRaw[ConfigKeyPassword],
		SecurityToken: cfgRaw[ConfigKeySecurityToken],
		PushTopicName: cfgRaw[ConfigKeyPushTopicName],
	}
	if cfg.Environment == "" {
		return Config{}, requiredConfigErr(ConfigKeyEnvironment)
	}
	if cfg.ClientId == "" {
		return Config{}, requiredConfigErr(ConfigKeyClientId)
	}
	if cfg.ClientSecret == "" {
		return Config{}, requiredConfigErr(ConfigKeyClientSecret)
	}
	if cfg.Username == "" {
		return Config{}, requiredConfigErr(ConfigKeyUsername)
	}
	if cfg.Password == "" {
		return Config{}, requiredConfigErr(ConfigKeyPassword)
	}
	if cfg.PushTopicName == "" {
		return Config{}, requiredConfigErr(ConfigKeyPushTopicName)
	}

	return cfg, nil
}

func requiredConfigErr(name string) error {
	return fmt.Errorf("%q config value must be set", name)
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Package config - exports for main

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configname = ".gatorconfig.json"

func Read() Config {
	targetdir := getPath()

	file, err := os.ReadFile(targetdir)
	if err != nil {
		return Config{}
	}

	newConfig := Config{}
	if err := json.Unmarshal(file, &newConfig); err != nil {
		return Config{}
	}

	return newConfig
}

func (c *Config) SetUser(name string) error {
	c.CurrentUserName = name
	targetdir := getPath()
	data, err := json.Marshal(&c)
	if err != nil {
		return err
	}

	if err := os.WriteFile(targetdir, data, 0o644); err != nil {
		return err
	}

	return nil
}

func getPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	targetdir := fmt.Sprintf("%s/%s", homedir, configname)
	fmt.Printf("targetdir:%s\n", targetdir)
	return targetdir
}

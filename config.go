package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	RenameRules     []GenreRenameRule `yaml:"GenreRenameRules"`
	authcompilation string            `yaml:"AuthCompilation"`
}

type GenreRenameRule struct {
	To   string `yaml:"T"`
	From string `yaml:"Fr"`
}

func readConfig(ConfigName string) (x *Config, err error) {
	var file []byte
	if file, err = ioutil.ReadFile(ConfigName); err != nil {
		return nil, err
	}
	x = new(Config)
	if err = yaml.Unmarshal(file, x); err != nil {
		return nil, err
	}
	// if x.LogLevel == "" {
	// 	x.LogLevel = "Debug"
	// }
	return x, nil
}

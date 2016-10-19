package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	// "github.com/ryanuber/go-glob"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Renamerules     []Match `yaml:"GenreRenameRules"`
	Authcompilation string  `yaml:"AuthCompilation"`
	DirectRules     []Match
	WildRules       []Match
}

type Match struct {
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
	parcerules(x)
	return x, nil
}

func parcerules(conf *Config) {
	for _, p := range conf.Renamerules {
		for _, f := range strings.Split(p.From, ",") {
			tempmatch := Match{To: p.To, From: strings.TrimSpace(f)}
			if strings.ContainsAny(tempmatch.From, "*") {
				conf.WildRules = append(conf.WildRules, tempmatch)
			} else {
				conf.DirectRules = append(conf.DirectRules, tempmatch)
			}
		}
	}
}

func (m Match) String() string {
	return fmt.Sprintf("%s -> %s", m.From, m.To)
}

// func (gs *GenreSubstitutions) Replace(genre string) string {
// 	for _, wild := range *gs {
// 		if glob.Glob(wild.From, genre) {
// 			return wild.To
// 		}
// 	}
// 	return ""
// }

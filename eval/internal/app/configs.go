package app

import (
	"fmt"
	"strings"
)

const (
	ConfigRegexOnly     = "regex-only"
	ConfigDictionary    = "dictionary"
	ConfigNER           = "ner"
	ConfigDictionaryNER = "dictionary+ner"
)

var AllConfigs = []string{ConfigRegexOnly, ConfigDictionary, ConfigNER, ConfigDictionaryNER}

var configAliases = map[string]string{
	"with-dictionary":    ConfigDictionary,
	"without-dictionary": ConfigRegexOnly,
	"regex":              ConfigRegexOnly,
	"ner-only":           ConfigNER,
	"ner+dictionary":     ConfigDictionaryNER,
}

func ResolveConfigs(configs, dictModes []string, nerURL string) ([]string, error) {
	ner := nerURL != ""
	if len(configs) == 0 {
		if len(dictModes) == 0 {
			dictModes = []string{"with", "without"}
		}
		for _, dm := range dictModes {
			switch {
			case dm == "with" && ner:
				configs = append(configs, ConfigDictionaryNER)
			case dm == "with":
				configs = append(configs, ConfigDictionary)
			case dm == "without" && ner:
				configs = append(configs, ConfigNER)
			case dm == "without":
				configs = append(configs, ConfigRegexOnly)
			default:
				return nil, fmt.Errorf("unknown --dictionary value %q (want with or without)", dm)
			}
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range configs {
		c = strings.ToLower(strings.TrimSpace(c))
		if a, ok := configAliases[c]; ok {
			c = a
		}
		switch c {
		case ConfigRegexOnly, ConfigDictionary:
		case ConfigNER, ConfigDictionaryNER:
			if !ner {
				return nil, fmt.Errorf("config %q needs --ner-url", c)
			}
		default:
			return nil, fmt.Errorf("unknown config %q (want one of %s)", c, strings.Join(AllConfigs, ", "))
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out, nil
}

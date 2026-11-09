package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const (
	RouteMasked      = "masked"
	RouteLocal       = "local"
	RoutePassthrough = "passthrough"
	RouteBlock       = "block"
)

type Config struct {
	Listen       string              `yaml:"listen"`
	Vault        VaultConfig         `yaml:"vault"`
	Upstreams    map[string]Upstream `yaml:"upstreams"`
	Policy       PolicyConfig        `yaml:"policy"`
	NER          NERConfig           `yaml:"ner"`
	Auth         AuthConfig          `yaml:"auth"`
	Log          LogConfig           `yaml:"log"`
	MaxBodyBytes int64               `yaml:"max_body_bytes"`

	dir string
}

type VaultConfig struct {
	Path    string `yaml:"path"`
	KeyEnv  string `yaml:"key_env"`
	KeyFile string `yaml:"key_file"`
}

type Upstream struct {
	BaseURL       string            `yaml:"base_url"`
	APIKeyEnv     string            `yaml:"api_key_env"`
	Model         string            `yaml:"model"`
	Headers       map[string]string `yaml:"headers"`
	HeaderTimeout Duration          `yaml:"header_timeout"`
}

type PolicyConfig struct {
	DefaultRoute        string            `yaml:"default_route"`
	MaskedUpstream      string            `yaml:"masked_upstream"`
	LocalUpstream       string            `yaml:"local_upstream"`
	PassthroughUpstream string            `yaml:"passthrough_upstream"`
	DateShift           *bool             `yaml:"date_shift"`
	DateShiftScope      string            `yaml:"date_shift_scope"`
	DateOrder           string            `yaml:"date_order"`
	ShiftMinDays        int               `yaml:"shift_min_days"`
	ShiftMaxDays        int               `yaml:"shift_max_days"`
	Dictionary          string            `yaml:"dictionary"`
	Entities            map[string]string `yaml:"entities"`
	Rules               []Rule            `yaml:"rules"`
	AllowRouteHeader    *bool             `yaml:"allow_route_header"`
	AllowScopeHeader    bool              `yaml:"allow_scope_header"`
	AllowImages         bool              `yaml:"allow_images"`
}

type Rule struct {
	Name       string   `yaml:"name"`
	MatchAny   []string `yaml:"match_any"`
	MatchRegex []string `yaml:"match_regex"`
	Entities   []string `yaml:"entities"`
	Route      string   `yaml:"route"`
	Upstream   string   `yaml:"upstream"`

	words   []*regexp.Regexp
	regexes []*regexp.Regexp
	types   []entity.Type
}

type NERConfig struct {
	URL       string            `yaml:"url"`
	Timeout   Duration          `yaml:"timeout"`
	Threshold float64           `yaml:"threshold"`
	Optional  bool              `yaml:"optional"`
	Labels    map[string]string `yaml:"labels"`
}

type AuthConfig struct {
	TokenEnv string `yaml:"token_env"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	v, err := time.ParseDuration(n.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", n.Value, err)
	}
	*d = Duration(v)
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, err
	}
	abs, _ := filepath.Abs(path)
	c.dir = filepath.Dir(abs)
	return c, nil
}

func Parse(data []byte) (*Config, error) {
	c := &Config{}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	return c, c.Validate()
}

func (c *Config) applyDefaults() {
	if env := strings.TrimSpace(os.Getenv("OUTWORLD_LISTEN")); env != "" {
		c.Listen = env
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8787"
	}
	if c.Vault.Path == "" {
		c.Vault.Path = "./outworld.vault"
	}
	if c.Vault.KeyEnv == "" && c.Vault.KeyFile == "" {
		c.Vault.KeyEnv = "OUTWORLD_VAULT_KEY"
	}
	if c.Policy.DefaultRoute == "" {
		c.Policy.DefaultRoute = RouteMasked
	}
	if c.Policy.DateShiftScope == "" {
		c.Policy.DateShiftScope = "global"
	}
	if c.Auth.TokenEnv == "" {
		c.Auth.TokenEnv = "OUTWORLD_TOKEN"
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 32 << 20
	}
}

func (c *Config) Resolve(p string) string {
	if p == "" || filepath.IsAbs(p) || c.dir == "" {
		return p
	}
	return filepath.Join(c.dir, p)
}

func validRoute(r string) bool {
	switch r {
	case RouteMasked, RouteLocal, RoutePassthrough, RouteBlock:
		return true
	}
	return false
}

func (c *Config) Validate() error {
	var errs []string
	for name, u := range c.Upstreams {
		if u.BaseURL == "" {
			errs = append(errs, fmt.Sprintf("upstreams.%s.base_url is required", name))
		}
	}
	if !validRoute(c.Policy.DefaultRoute) {
		errs = append(errs, fmt.Sprintf("policy.default_route %q is not one of masked, local, passthrough, block", c.Policy.DefaultRoute))
	}
	checkUp := func(field, name string) {
		if name == "" {
			return
		}
		if _, ok := c.Upstreams[name]; !ok {
			errs = append(errs, fmt.Sprintf("%s refers to unknown upstream %q", field, name))
		}
	}
	checkUp("policy.masked_upstream", c.Policy.MaskedUpstream)
	checkUp("policy.local_upstream", c.Policy.LocalUpstream)
	checkUp("policy.passthrough_upstream", c.Policy.PassthroughUpstream)
	if c.Policy.MaskedUpstream == "" && len(c.Upstreams) > 0 {
		errs = append(errs, "policy.masked_upstream is required when upstreams are configured")
	}
	for k, v := range c.Policy.Entities {
		if _, err := entity.ParseType(k); err != nil {
			errs = append(errs, "policy.entities: "+err.Error())
		}
		if _, err := entity.ParseAction(v); err != nil {
			errs = append(errs, fmt.Sprintf("policy.entities.%s: %v", k, err))
		}
	}
	switch strings.ToLower(c.Policy.DateOrder) {
	case "", "mdy", "dmy":
	default:
		errs = append(errs, "policy.date_order must be mdy or dmy")
	}
	for i := range c.Policy.Rules {
		r := &c.Policy.Rules[i]
		if !validRoute(r.Route) {
			errs = append(errs, fmt.Sprintf("policy.rules[%d].route %q is invalid", i, r.Route))
		}
		checkUp(fmt.Sprintf("policy.rules[%d].upstream", i), r.Upstream)
		if len(r.MatchAny)+len(r.MatchRegex)+len(r.Entities) == 0 {
			errs = append(errs, fmt.Sprintf("policy.rules[%d] has no match_any, match_regex or entities", i))
		}
		r.words, r.regexes, r.types = nil, nil, nil
		for _, w := range r.MatchAny {
			r.words = append(r.words, regexp.MustCompile(`(?i)(?:^|\W)`+regexp.QuoteMeta(strings.TrimSpace(w))+`(?:\W|$)`))
		}
		for _, p := range r.MatchRegex {
			re, err := regexp.Compile(p)
			if err != nil {
				errs = append(errs, fmt.Sprintf("policy.rules[%d].match_regex: %v", i, err))
				continue
			}
			r.regexes = append(r.regexes, re)
		}
		for _, t := range r.Entities {
			et, err := entity.ParseType(t)
			if err != nil {
				errs = append(errs, fmt.Sprintf("policy.rules[%d].entities: %v", i, err))
				continue
			}
			r.types = append(r.types, et)
		}
	}
	if len(errs) > 0 {
		return errors.New("invalid config:\n  " + strings.Join(errs, "\n  "))
	}
	return nil
}

func (r *Rule) MatchText(text string) bool {
	for _, re := range r.words {
		if re.MatchString(text) {
			return true
		}
	}
	for _, re := range r.regexes {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

func (r *Rule) EntityTypes() []entity.Type { return r.types }

func (c *Config) EnginePolicy() (outworld.Policy, error) {
	p := outworld.DefaultPolicy()
	for k, v := range c.Policy.Entities {
		t, err := entity.ParseType(k)
		if err != nil {
			return p, err
		}
		a, err := entity.ParseAction(v)
		if err != nil {
			return p, err
		}
		p.Actions[t] = a
	}
	if c.Policy.DateShift != nil {
		p.DateShift = *c.Policy.DateShift
	}
	if strings.EqualFold(c.Policy.DateOrder, "dmy") {
		p.DateOrder = dateshift.DMY
	}
	if c.Policy.ShiftMinDays > 0 {
		p.ShiftMinDays = c.Policy.ShiftMinDays
	}
	if c.Policy.ShiftMaxDays > 0 {
		p.ShiftMaxDays = c.Policy.ShiftMaxDays
	}
	return p, nil
}

func (c *Config) RouteHeaderAllowed() bool {
	return c.Policy.AllowRouteHeader == nil || *c.Policy.AllowRouteHeader
}

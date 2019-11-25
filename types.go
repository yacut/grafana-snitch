package main


// Config definition
type Config struct {
	Rules RuleConfigs `yaml:"rules"`
	Mode  string      `yaml:"mode"`
}

// RuleConfigs definition
type RuleConfigs struct {
	Groups []Rule `yaml:"groups"`
	Users  []Rule `yaml:"users"`
}

// Rule definition
type Rule struct {
	Name         string `yaml:"name"`
	Email        string `yaml:"email"`
	Organization string `yaml:"organization"`
	Role         string `yaml:"role"`

}
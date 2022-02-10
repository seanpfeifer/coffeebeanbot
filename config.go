package coffeebeanbot

import "github.com/BurntSushi/toml"

// Config is the Bot's configuration data
type Config struct {
	CmdPrefix    string `toml:"cmdPrefix"`    // The prefix the bot will look for in chat before all known commands
	WorkEndAudio string `toml:"workEndAudio"` // The DCA audio file that will be played when a Pomodoro ends. This is only played if the user is in voice chat in the Discord Server (Guild).
}

// Secrets is the Bot's per-user data, some of which is secret
type Secrets struct {
	AuthToken string `toml:"authToken"` // AuthToken is all that we need to authenticate with Discord as the bot's user
	AppID     string `toml:"appID"`     // The application ID from the bot info. This isn't necessarily "secret"
}

// LoadConfigFile loads the config from the given path, returning the config or an error if one occurred.
// I generally prefer config files over environment variables, due to the ease of setting them up as secrets
// in Kubernetes.
func LoadConfigFile(path string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)

	return &cfg, err
}

// LoadSecretsFile loads the Discord secrets from the given path, returning the Secrets or an error if one occurred.
func LoadSecretsFile(path string) (*Secrets, error) {
	var secrets Secrets
	_, err := toml.DecodeFile(path, &secrets)

	return &secrets, err
}

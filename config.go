package coffeebeanbot

import (
	"encoding/json"
	"io/ioutil"
)

// Config is the Bot's configuration data
type Config struct {
	CmdPrefix     string `json:"cmdPrefix"`     // The prefix the bot will look for in chat before all known commands
	WorkEndAudio  string `json:"workEndAudio"`  // The DCA audio file that will be played when a Pomodoro ends. This is only played if the user is in voice chat in the Discord Server (Guild).
	EnableMetrics bool   `json:"enableMetrics"` // True if metrics exporting should be enabled, false otherwise
	DebugMetrics  bool   `json:"debugMetrics"`  // True if metrics reporting (when enabled) should ONLY go to stdout, false otherwise. If EnabledMetrics is false, this does nothing.
}

// Secrets is the Bot's per-user data, some of which is secret
type Secrets struct {
	AuthToken string `json:"authToken"` // AuthToken is all that we need to authenticate with Discord as the bot's user
	ClientID  string `json:"clientID"`  // Used to create the invite link for the bot - this isn't necessary for Discord login, nor does it need to be "secret"
}

// LoadConfigFile loads the config from the given path, returning the config or an error if one occurred.
// I generally prefer config files over environment variables, due to the ease of setting them up as secrets
// in Kubernetes.
func LoadConfigFile(path string) (*Config, error) {
	var cfg Config
	err := parseJSONFile(path, &cfg)

	return &cfg, err
}

// LoadSecretsFile loads the Discord secrets from the given path, returning the Secrets or an error if one occurred.
func LoadSecretsFile(path string) (*Secrets, error) {
	var secrets Secrets
	err := parseJSONFile(path, &secrets)

	return &secrets, err
}

// parseJSONFile is a common pattern for reading + unmarshalling a JSON file into a struct.
func parseJSONFile(path string, v interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

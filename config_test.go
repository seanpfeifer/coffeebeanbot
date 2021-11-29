package coffeebeanbot

import (
	"testing"

	. "github.com/seanpfeifer/rigging/assert"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfigFile("./cfg.toml")
	ExpectedActual(t, nil, err, "loading config file")
	ExpectedActual(t, "!cbb ", cfg.CmdPrefix, "command prefix")
	ExpectedActual(t, "./audio/airhorn.dca", cfg.WorkEndAudio, "work end audio")
}

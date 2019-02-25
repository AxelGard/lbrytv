package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Settings stores project settings such as download paths, host prefix for URLs and so on
var Settings = viper.New()

var overriddenValues map[string]interface{}

func init() {
	read()
}

// Read parses `lbryweb.yml`
func read() {
	Settings.SetEnvPrefix("LW")
	Settings.BindEnv("Debug")
	Settings.SetDefault("Debug", false)
	Settings.BindEnv("Lbrynet")
	Settings.SetDefault("Lbrynet", "http://localhost:5279/")

	Settings.SetDefault("Address", ":8080")
	Settings.SetDefault("Host", "http://localhost:8080")
	Settings.SetDefault("BaseContentURL", "http://localhost:8080/content/")

	Settings.SetDefault("StaticURLPrefix", "/static/")
	Settings.SetDefault("StaticDir", "./assets/static")
	Settings.SetConfigName("lbryweb") // name of config file (without extension)
	Settings.AddConfigPath("./")
	Settings.AddConfigPath("../")
	Settings.AddConfigPath("$HOME/.lbryweb")
	err := Settings.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error reading config file: %s", err))
	}
	overriddenValues = make(map[string]interface{})
}

// IsProduction is true if we are running in a production environment
func IsProduction() bool {
	return !Settings.GetBool("Debug")
}

func ProjectRoot() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}

func Override(key string, value interface{}) {
	overriddenValues[key] = Settings.Get(key)
	Settings.Set(key, value)
}

func RestoreOverridden() {
	if len(overriddenValues) == 0 {
		return
	}
	for k, v := range overriddenValues {
		Settings.Set(k, v)
	}
	overriddenValues = make(map[string]interface{})
}

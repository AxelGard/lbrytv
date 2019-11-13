package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverride(t *testing.T) {
	c := GetConfig()
	originalSetting := c.Viper.Get("Lbrynet")
	Override("Lbrynet", "http://www.google.com:8080/api/proxy")
	assert.Equal(t, "http://www.google.com:8080/api/proxy", c.Viper.Get("Lbrynet"))
	RestoreOverridden()
	assert.Equal(t, originalSetting, c.Viper.Get("Lbrynet"))
	assert.Empty(t, overriddenValues)
}

func TestIsProduction(t *testing.T) {
	Override("Debug", false)
	assert.True(t, IsProduction())
	Override("Debug", true)
	assert.False(t, IsProduction())
	defer RestoreOverridden()
}

func TestGetAllLbrynets(t *testing.T) {
	assert.Equal(t, map[string]string{
		"default": "http://lbrynet1:5279/",
		"sdk1":    "http://lbrynet2:5279/",
		"sdk2":    "http://lbrynet3:5279/",
	}, GetAllLbrynets())
}

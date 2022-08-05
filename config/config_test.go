package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()

	cfgPath := filepath.Join(t.TempDir(), ConfigFile)
	assert.NoError(t, WriteConfig(cfgPath, cfg))

	res, err := ReadConfig(cfgPath)
	assert.NoError(t, err)
	assert.Equal(t, cfg, res)
}

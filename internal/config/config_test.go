package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	loader := NewConfigLoader()
	_, err := loader.Load()
	assert.NoError(t, err)
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatEnv(t *testing.T) {
	require := require.New(t)

	envStrings := []string{
		"FOO=BAR",
		"BAZ=",
		"QUX",
		"",
		"QUZZ=EMBEDDED=EQUALS",
	}

	env := formatEnv(envStrings)

	expected := map[string]string{
		"FOO":  "BAR",
		"BAZ":  "",
		"QUX":  "",
		"QUZZ": "EMBEDDED=EQUALS",
	}
	require.Equal(expected, env)
}

func TestFormatEnv_EmpytSet(t *testing.T) {
	assert := assert.New(t)

	env := formatEnv(make([]string, 0))
	assert.NotNil(env)
	assert.Empty(env)
}

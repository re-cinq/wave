package tui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	assert.True(t, IsProcessAlive(pid), "current process should be alive")
}

func TestIsProcessAlive_ZeroPID(t *testing.T) {
	assert.False(t, IsProcessAlive(0), "PID 0 should return false")
}

func TestIsProcessAlive_NegativePID(t *testing.T) {
	assert.False(t, IsProcessAlive(-1), "negative PID should return false")
}

func TestIsProcessAlive_VeryLargePID(t *testing.T) {
	// PID 4194304 is beyond the default Linux PID max (32768 or 4194304 on 64-bit)
	// and extremely unlikely to exist.
	assert.False(t, IsProcessAlive(99999999), "very large PID should return false")
}

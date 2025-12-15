package utils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRealIP(t *testing.T) {
	r, err := http.NewRequest("GET", "/support", strings.NewReader(""))
	require.NoError(t, err)

	// no ip
	ip := GetRealIP(r)
	assert.Equal(t, "", ip)

	// ip
	expectedIP := "135.12.64.32"
	r.Header.Set("x-real-ip", expectedIP)
	ip = GetRealIP(r)
	assert.Equal(t, expectedIP, ip)
}

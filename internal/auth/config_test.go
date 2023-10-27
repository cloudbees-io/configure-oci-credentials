package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchConfig(t *testing.T) {
	type Test struct {
		registry string
		expected bool
	}

	tt := []Test{

		{registry: "https://docker.io.bad/", expected: false},
		{registry: "https://docker.io", expected: true},
		{registry: "docker.io/", expected: true},
		{registry: "docker.io", expected: true},
		{registry: "https://docker.io/", expected: true},
		{registry: "https://index.docker.io.bad/", expected: false},
		{registry: "https://index.docker.io", expected: true},
		{registry: "index.docker.io/", expected: true},
		{registry: "index.docker.io", expected: true},
		{registry: "https://index.docker.io/", expected: true},
	}

	for _, e := range tt {
		actual := isDockerHubRegistry(e.registry)
		require.Equal(t, e.expected, actual)
	}

}

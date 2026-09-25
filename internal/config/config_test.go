package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadValidation(t *testing.T) {
	cases := []struct {
		name      string
		env       map[string]string
		wantError string
	}{
		{
			name: "a key and an address are all the service needs beyond the defaults",
			env: map[string]string{
				"MESSAGEBIRD_ACCESS_KEY": "live-key",
				"MESSAGEBIRD_BASE_URL":   "https://provider.example",
			},
		},
		{
			name:      "an absent provider key is rejected",
			env:       map[string]string{"MESSAGEBIRD_BASE_URL": "https://provider.example"},
			wantError: "MESSAGEBIRD_ACCESS_KEY",
		},
		{
			name: "a blank provider key is rejected too, since compose passes one through",
			env: map[string]string{
				"MESSAGEBIRD_ACCESS_KEY": "",
				"MESSAGEBIRD_BASE_URL":   "https://provider.example",
			},
			wantError: "MESSAGEBIRD_ACCESS_KEY",
		},
		{
			name:      "an absent provider address is rejected, so no forgotten variable reaches production",
			env:       map[string]string{"MESSAGEBIRD_ACCESS_KEY": "live-key"},
			wantError: "MESSAGEBIRD_BASE_URL",
		},
		{
			name: "a blank provider address is rejected too",
			env: map[string]string{
				"MESSAGEBIRD_ACCESS_KEY": "live-key",
				"MESSAGEBIRD_BASE_URL":   "",
			},
			wantError: "MESSAGEBIRD_BASE_URL",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := load[Config](c.env)

			if c.wantError == "" {
				require.NoError(t, err, "this environment is valid and load must accept it")
				return
			}
			require.Error(t, err, "load must reject this environment")
			require.Contains(t, err.Error(), c.wantError,
				"the error must name the variable at fault so the operator can fix it")
		})
	}
}

func TestLoadDatabaseIgnoresTheProviderKey(t *testing.T) {
	d, err := load[Database](map[string]string{})
	require.NoError(t, err)
	require.Equal(t, "localhost", d.Host)
}

func TestLoadAppliesDefaults(t *testing.T) {
	c, err := load[Config](map[string]string{
		"MESSAGEBIRD_ACCESS_KEY": "live-key",
		"MESSAGEBIRD_BASE_URL":   "https://provider.example",
	})
	require.NoError(t, err)

	require.Equal(t, ":8080", c.HTTPAddr, "the default listen address")
	require.Equal(t, "ACME", c.MessageBird.Originator, "the default originator")
	require.Equal(t, 10*time.Second, c.MessageBird.Timeout, "outbound calls must always carry a timeout")

	require.Equal(t, "localhost", c.Database.Host, "the defaults must point at the database docker compose starts")
	require.Equal(t, uint16(5432), c.Database.Port)
	require.Equal(t, "notifications", c.Database.User)
	require.Equal(t, "notifications", c.Database.Password)
	require.Equal(t, "notifications", c.Database.Name)
	require.Equal(t, "disable", c.Database.SSLMode)
	require.Equal(t, 10*time.Second, c.Database.ConnectTimeout, "a connection attempt must always carry a timeout")
}

func TestLoadReadsEachDatabaseParameter(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		value  string
		assert func(t *testing.T, d Database)
	}{
		{
			name: "host", key: "DB_HOST", value: "db.internal",
			assert: func(t *testing.T, d Database) { require.Equal(t, "db.internal", d.Host) },
		},
		{
			name: "port", key: "DB_PORT", value: "6432",
			assert: func(t *testing.T, d Database) { require.Equal(t, uint16(6432), d.Port) },
		},
		{
			name: "user", key: "DB_USER", value: "app",
			assert: func(t *testing.T, d Database) { require.Equal(t, "app", d.User) },
		},
		{
			name: "password", key: "DB_PASSWORD", value: "s3cret",
			assert: func(t *testing.T, d Database) { require.Equal(t, "s3cret", d.Password) },
		},
		{
			name: "database name", key: "DB_NAME", value: "prod",
			assert: func(t *testing.T, d Database) { require.Equal(t, "prod", d.Name) },
		},
		{
			name: "ssl mode", key: "DB_SSLMODE", value: "require",
			assert: func(t *testing.T, d Database) { require.Equal(t, "require", d.SSLMode) },
		},
		{
			name: "connect timeout", key: "DB_CONNECT_TIMEOUT", value: "3s",
			assert: func(t *testing.T, d Database) { require.Equal(t, 3*time.Second, d.ConnectTimeout) },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := load[Database](map[string]string{c.key: c.value})
			require.NoError(t, err)

			c.assert(t, d)
		})
	}
}

func TestLoadRejectsAPortThatIsNotAPort(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "not a number", value: "not-a-port"},
		{name: "beyond the port range", value: "70000"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := load[Database](map[string]string{"DB_PORT": c.value})

			require.Error(t, err)
			require.Contains(t, err.Error(), "Port", "the error must name the field at fault")
			require.Contains(t, err.Error(), c.value, "and quote the value that could not be parsed")
		})
	}
}

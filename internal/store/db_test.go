package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notifications/internal/config"
)

func TestPoolConfigCarriesEveryDatabaseField(t *testing.T) {
	cfg, err := poolConfig(config.Database{
		Host:           "db.internal",
		Port:           6432,
		User:           "app",
		Password:       "s3cret",
		Name:           "prod",
		SSLMode:        "disable",
		ConnectTimeout: 3 * time.Second,
	})
	require.NoError(t, err)

	conn := cfg.ConnConfig
	require.Equal(t, "db.internal", conn.Host)
	require.Equal(t, uint16(6432), conn.Port)
	require.Equal(t, "app", conn.User)
	require.Equal(t, "s3cret", conn.Password)
	require.Equal(t, "prod", conn.Database)
	require.Equal(t, 3*time.Second, conn.ConnectTimeout, "a connection attempt must always carry a timeout")
	require.Nil(t, conn.TLSConfig, "sslmode=disable means no TLS")
}

func TestPoolConfigReadsTheSSLMode(t *testing.T) {
	cfg, err := poolConfig(config.Database{Host: "localhost", Port: 5432, SSLMode: "require"})
	require.NoError(t, err)

	require.NotNil(t, cfg.ConnConfig.TLSConfig, "sslmode=require must produce a TLS config")
}

func TestPoolConfigRejectsAnSSLModeThatIsNotOne(t *testing.T) {
	_, err := poolConfig(config.Database{Host: "localhost", Port: 5432, SSLMode: "sort-of"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "DB_SSLMODE", "the error must name the variable at fault")
	require.Contains(t, err.Error(), "sort-of", "and quote the value that could not be parsed")
}

func TestPoolConfigTakesCredentialsVerbatim(t *testing.T) {
	cfg, err := poolConfig(config.Database{
		Host:     "localhost",
		Port:     5432,
		User:     "no@body",
		Password: `p@ss:w/rd?x y'z\`,
		Name:     "notifications",
		SSLMode:  "disable",
	})
	require.NoError(t, err)

	require.Equal(t, "no@body", cfg.ConnConfig.User)
	require.Equal(t, `p@ss:w/rd?x y'z\`, cfg.ConnConfig.Password)
	require.Equal(t, "localhost", cfg.ConnConfig.Host)
	require.Equal(t, "notifications", cfg.ConnConfig.Database)
}

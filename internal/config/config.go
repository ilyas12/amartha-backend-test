package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	AppPort string

	MySQLHost string
	MySQLPort string
	MySQLDB   string
	MySQLUser string
	MySQLPass string

	RedisAddr string
	RedisDB   int

	IdempTTLSecs int
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Load() *Config {
	c := &Config{
		AppPort:   getenv("APP_PORT", "8080"),
		MySQLHost: getenv("MYSQL_HOST", "mysql"),
		MySQLPort: getenv("MYSQL_PORT", "3306"),
		MySQLDB:   getenv("MYSQL_DB", "amartha"),
		MySQLUser: getenv("MYSQL_USER", "amartha"),
		MySQLPass: getenv("MYSQL_PASS", "amartha"),

		RedisAddr:    getenv("REDIS_ADDR", "redis:6379"),
		IdempTTLSecs: 300,
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RedisDB = n
		}
	}
	if v := os.Getenv("IDEMPOTENCY_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.IdempTTLSecs = n
		}
	}
	return c
}

func (c *Config) Validate() error {
	if c.MySQLHost == "" || c.MySQLPort == "" || c.MySQLDB == "" || c.MySQLUser == "" {
		return errors.New("missing MySQL config (MYSQL_HOST/PORT/DB/USER)")
	}
	// ensure port is valid
	if _, err := net.LookupPort("tcp", c.MySQLPort); err != nil {
		return fmt.Errorf("invalid MYSQL_PORT %q: %w", c.MySQLPort, err)
	}
	if c.AppPort == "" {
		return errors.New("missing APP_PORT")
	}
	return nil
}

func (c *Config) mysqlAddr() string { return net.JoinHostPort(c.MySQLHost, c.MySQLPort) }

func (c *Config) MySQLDSN() string {
	// multiStatements=true is handy for migrations; parseTime needed for DATETIME
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true&parseTime=true&charset=utf8mb4,utf8",
		c.MySQLUser, c.MySQLPass, c.mysqlAddr(), c.MySQLDB)
}

package config

import (
	"fmt"
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
		AppPort:      getenv("APP_PORT", ""),
		MySQLHost:    getenv("MYSQL_HOST", ""),
		MySQLPort:    getenv("MYSQL_PORT", ""),
		MySQLDB:      getenv("MYSQL_DB", ""),
		MySQLUser:    getenv("MYSQL_USER", ""),
		MySQLPass:    getenv("MYSQL_PASS", ""),
		RedisAddr:    getenv("REDIS_ADDR", ""),
		IdempTTLSecs: 172800,
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

func (c *Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4,utf8",
		c.MySQLUser, c.MySQLPass, c.MySQLHost, c.MySQLPort, c.MySQLDB)
}

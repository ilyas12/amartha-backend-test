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
		AppPort:      getenv("APP_PORT", "8080"),
		MySQLHost:    getenv("MYSQL_HOST", "127.0.0.1"),
		MySQLPort:    getenv("MYSQL_PORT", "3306"),
		MySQLDB:      getenv("MYSQL_DB", "app"),
		MySQLUser:    getenv("MYSQL_USER", "app"),
		MySQLPass:    getenv("MYSQL_PASS", "app"),
		RedisAddr:    getenv("REDIS_ADDR", "127.0.0.1:6379"),
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

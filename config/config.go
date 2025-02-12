package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type PostgresConfig struct {
	User     string
	Password string
	Port     string
	Host     string
	DBName   string
}

type MailerConfig struct {
	Email              string
	Password           string
	AdditionalEmail    string
	AdditionalPassword string
}

type FileConfig struct {
	RootPath string
}

type TimeoutsConfig struct {
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	ContextTimeout time.Duration
}

var (
	Lokle        ServerConfig
	RedisSession RedisConfig
	RedisUser    RedisConfig
	Postgres     PostgresConfig
	Mailer       MailerConfig
	Timeouts     TimeoutsConfig
	File         FileConfig
)

func SetConfig() {
	viper.SetConfigFile("config.json")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	Lokle = ServerConfig{
		Port: viper.GetString(`lokle.port`),
	}

	RedisSession = RedisConfig{
		Addr:     viper.GetString(`redis.address`),
		Password: viper.GetString(`redis.password`),
		DB:       viper.GetInt(`redis.session_db_name`),
	}

	RedisUser = RedisConfig{
		Addr:     viper.GetString(`redis.address`),
		Password: viper.GetString(`redis.password`),
		DB:       viper.GetInt(`redis.user_db_name`),
	}

	Postgres = PostgresConfig{
		Port:     viper.GetString(`postgres.port`),
		Host:     viper.GetString(`postgres.host`),
		User:     viper.GetString(`postgres.user`),
		Password: viper.GetString(`postgres.pass`),
		DBName:   viper.GetString(`postgres.name`),
	}

	Mailer = MailerConfig{
		Email:              viper.GetString(`mailer.email`),
		Password:           viper.GetString(`mailer.password`),
		AdditionalEmail:    viper.GetString(`mailer.additional_email`),
		AdditionalPassword: viper.GetString(`mailer.additional_password`),
	}

	File = FileConfig{
		RootPath: viper.GetString(`file.root_path`),
	}

	Timeouts = TimeoutsConfig{
		WriteTimeout:   5 * time.Second,
		ReadTimeout:    5 * time.Second,
		ContextTimeout: time.Second * 2,
	}
}

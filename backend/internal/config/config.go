package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构体
type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
	Log      LogConfig      `yaml:"log"`
	Mail     MailConfig     `yaml:"mail"`
	SMS      SMSConfig      `yaml:"sms"`
	SSO      SSOConfig      `yaml:"sso"`
	ICPAPI   ICPAPIConfig   `yaml:"icp_api"`
	Crawler  CrawlerConfig  `yaml:"crawler"`
	Storage  StorageConfig  `yaml:"storage"`
	Frontend FrontendConfig `yaml:"frontend"`
}

type AppConfig struct {
	Name   string `yaml:"name"`
	Port   int    `yaml:"port"`
	Mode   string `yaml:"mode"`
	Secret string `yaml:"secret"`
}

type DatabaseConfig struct {
	Driver       string `yaml:"driver"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DBName       string `yaml:"dbname"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	LogLevel     string `yaml:"log_level"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type JWTConfig struct {
	Secret       string `yaml:"secret"`
	ExpireHours  int    `yaml:"expire_hours"`
	Issuer       string `yaml:"issuer"`
}

type LogConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	Path      string `yaml:"path"`
	MaxSize   int    `yaml:"max_size"` // MB
	MaxBackups int   `yaml:"max_backups"`
	MaxAge    int    `yaml:"max_age"` // days
	Compress  bool   `yaml:"compress"`
}

type MailConfig struct {
	Enabled   bool   `yaml:"enabled"`
	SMTPHost  string `yaml:"smtp_host"`
	SMTPPort  int    `yaml:"smtp_port"`
	SMTPUser  string `yaml:"smtp_user"`
	SMTPPass  string `yaml:"smtp_pass"`
	FromName  string `yaml:"from_name"`
	FromAddr  string `yaml:"from_addr"`
}

type SMSConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Provider    string `yaml:"provider"`
	AccessKey   string `yaml:"access_key"`
	SecretKey   string `yaml:"secret_key"`
	SignName    string `yaml:"sign_name"`
	TemplateCode string `yaml:"template_code"`
}

type SSOConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Provider          string `yaml:"provider"`
	ClientID          string `yaml:"client_id"`
	ClientSecret      string `yaml:"client_secret"`
	RedirectURI       string `yaml:"redirect_uri"`
	AuthURL           string `yaml:"auth_url"`
	TokenURL          string `yaml:"token_url"`
	UserInfoURL       string `yaml:"user_info_url"`
	DefaultRoleID     uint   `yaml:"default_role_id"`
	AllowPasswordLogin bool  `yaml:"allow_password_login"`
}

type ICPAPIConfig struct {
	Key string `yaml:"key"`
	URL string `yaml:"url"`
}

type CrawlerConfig struct {
	APIHost string `yaml:"api_host"`
	APIPort int    `yaml:"api_port"`
	Timeout int    `yaml:"timeout"`
}
func (c *CrawlerConfig) APIBaseURL() string {
	return fmt.Sprintf("%s:%d", c.APIHost, c.APIPort)
}

type StorageConfig struct {
	Type             string   `yaml:"type"`
	Path             string   `yaml:"path"`
	MaxUploadSize    int64    `yaml:"max_upload_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
}

type FrontendConfig struct {
	URL string `yaml:"url"`
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 支持环境变量覆盖
	loadEnvOverrides(&cfg)

	return &cfg, nil
}

// loadEnvOverrides 从环境变量加载覆盖配置
func loadEnvOverrides(cfg *Config) {
	if host := os.Getenv("MYSQL_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if port := os.Getenv("MYSQL_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Database.Port)
	}
	if user := os.Getenv("MYSQL_USER"); user != "" {
		cfg.Database.Username = user
	}
	if pass := os.Getenv("MYSQL_PASSWORD"); pass != "" {
		cfg.Database.Password = pass
	}
	if dbname := os.Getenv("MYSQL_DATABASE"); dbname != "" {
		cfg.Database.DBName = dbname
	}

	if host := os.Getenv("REDIS_HOST"); host != "" {
		cfg.Redis.Host = host
	}
	if port := os.Getenv("REDIS_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Redis.Port)
	}

	if port := os.Getenv("APP_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.App.Port)
	}
}

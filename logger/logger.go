package logger

import (
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/delichik/daf/config"
)

const (
	ModuleName = "logger"
)

var defaultLoggerConfig = &Config{
	Level:     "info",
	Format:    "text",
	LogPath:   "stdout",
	MaxSize:   50,
	MaxBackup: 1,
	MaxAge:    30,
	Compress:  true,
	logLevel:  zapcore.InfoLevel,
}

func init() {
	config.RegisterModuleConfig(ModuleName, defaultLoggerConfig)
}

type Config struct {
	Level     string `yaml:"level" comment:"Min log output level"`
	Format    string `yaml:"format" comment:"Log output format: json|text"`
	LogPath   string `yaml:"log-path" comment:"Path to write log, use \"stdout\" to write to console"`
	MaxSize   int    `yaml:"max-size" comment:"Maximum size (MB) of a log file"`
	MaxBackup int    `yaml:"max-backup" comment:"Maximum count of log backup"`
	MaxAge    int    `yaml:"max-age" comment:"Maximum saving days of a log backup"`
	Compress  bool   `yaml:"compress" comment:"Compress the backups"`

	LogDriver  zapcore.WriteSyncer `yaml:"-"`
	logLevel   zapcore.Level       `yaml:"-"`
	ModePrefix string              `yaml:"-"`
}

func (c *Config) Check() error {
	if c.Level != "" {
		err := c.logLevel.UnmarshalText([]byte(c.Level))
		if err != nil {
			return fmt.Errorf("level: %w", err)
		}
	}

	if c.Format != "" && c.Format != "json" && c.Format != "text" {
		return errors.New(`format: must be "json" or "text"`)
	}

	if len(c.LogPath) == 0 && c.LogDriver == nil {
		return errors.New(`log_path: required`)
	}

	if c.LogPath != "stdout" && c.LogDriver == nil {
		if c.MaxSize <= 0 {
			return errors.New(`max_size: must grater than 0`)
		}

		if c.MaxBackup < 0 {
			return errors.New(`max_backup: must grater than or equal to 0`)
		}

		if c.MaxAge <= 0 {
			return errors.New(`max_backup: must grater than 0`)
		}
	}
	return nil
}

func (c *Config) Clone() config.ModuleConfig {
	newObj := *c
	newObj.logLevel = c.logLevel
	return &newObj
}

func (c *Config) Compare(config.ModuleConfig) bool {
	return true
}

var defaultLogger *zap.Logger

var loggers = make(map[string]*zap.Logger)

func InitDefaultManual(loggerConfig *Config) {
	defaultLogger = doInit("", loggerConfig)
}

func InitDefault(c config.ConfigSet) {
	defaultLogger = Init("", c)
}

func RegisterAdditionalLogger(moduleName string) {
	if moduleName == "" {
		return
	}

	cfg := defaultLoggerConfig.Clone().(*Config)
	cfg.LogPath = "logs/" + moduleName + ".log"
	name := moduleName + "-" + ModuleName
	config.RegisterModuleConfig(name, cfg)
	return
}

func Init(moduleName string, c config.ConfigSet) *zap.Logger {
	if moduleName == "" {
		moduleName = ModuleName
	} else {
		moduleName = moduleName + "-" + ModuleName
	}
	var loggerConfig *Config
	t := c.GetModuleConfig(moduleName)
	if t == nil || t.(*Config) == nil {
		loggerConfig = defaultLoggerConfig
	} else {
		loggerConfig = t.(*Config)
	}
	return doInit(moduleName, loggerConfig)
}

func doInit(moduleName string, loggerConfig *Config) *zap.Logger {
	var writeSyncer zapcore.WriteSyncer
	if loggerConfig.LogDriver != nil {
		writeSyncer = loggerConfig.LogDriver
	} else if loggerConfig.LogPath == "stdout" {
		writeSyncer = os.Stdout
	} else {
		writeSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   loggerConfig.LogPath,
			MaxSize:    loggerConfig.MaxSize,
			MaxBackups: loggerConfig.MaxBackup,
			MaxAge:     loggerConfig.MaxAge,
			Compress:   loggerConfig.Compress,
		})
	}
	var core zapcore.Core
	if loggerConfig.Format == "json" {
		core = zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), writeSyncer, loggerConfig.logLevel)
	} else if loggerConfig.Format == "text" {
		core = zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), writeSyncer, loggerConfig.logLevel)
	}
	zapcore.TimeEncoderOfLayout(time.RFC3339)
	l := zap.New(core, zap.AddStacktrace(zap.ErrorLevel), zap.WithCaller(true), zap.AddCallerSkip(1))
	loggers[moduleName] = l
	return l
}

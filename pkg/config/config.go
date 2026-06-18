/**
@Time : 2026/01/15 15:15
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package config

import (
	_ "embed"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"os"
	"strconv"
)

// 全局配置变量
var Conf = new(Config)

// 全局配置结构体
type Config struct {
	System    *SystemConfig    `mapstructure:"system" json:"system"`
	Logs      *Logs            `mapstructure:"logs" json:"logs"`
	Database  *Database        `mapstructure:"database" json:"database"`
	Mysql     *MysqlConfig     `mapstructure:"mysql" json:"mysql"`
	Sqlite    *SqliteConfig    `mapstructure:"sqlite" json:"sqlite"`
	Jwt       *JwtConfig       `mapstructure:"jwt" json:"jwt"`
	RateLimit *RateLimitConfig `mapstructure:"rate-limit" json:"rateLimit"`
	Redis     *RedisConfig     `mapstructure:"redis" json:"redis"`
	Onnx      *OnnxConfig      `mapstructure:"onnx" json:"onnx"`
	Plc       *PlcConfig       `mapstructure:"plc" json:"plc"`
}

// 系统配置
type SystemConfig struct {
	Mode              string      `mapstructure:"mode" json:"mode"`
	UrlPathPrefix     string      `mapstructure:"url-path-prefix" json:"urlPathPrefix"`
	Host              string      `mapstructure:"host" json:"host"`
	Port              int         `mapstructure:"port" json:"port"`
	IsAnalysis        bool        `mapstructure:"isAnalysis" json:"isAnalysis"`               // 是否启用分析
	EnableRecording   bool        `mapstructure:"enableRecording" json:"enableRecording"`     // 是否启用告警录像
	IsStartOPlc       bool        `mapstructure:"isStartOPlc" json:"isStartOPlc"`             // 是否启用plc
	SnapshotRootPath  string      `mapstructure:"snapshotRootPath" json:"snapshotRootPath"`   // 快照图片根路径
	RecordingRootPath string      `mapstructure:"recordingRootPath" json:"recordingRootPath"` // 录像文件根路径
	DevMode           bool        `mapstructure:"devMode" json:"devMode"`                     // 开发模式（true=源码路径，false=程序执行目录）
	Task              *TaskConfig `mapstructure:"task" json:"task"`                           // 定时任务配置
}

// 定时任务配置
type TaskConfig struct {
	SnapshotRetentionDays int    `mapstructure:"snapshot-retention-days"` // 快照保留天数
	RecordRetentionDays   int    `mapstructure:"record-retention-days"`   // 录像保留天数
	CronExpression        string `mapstructure:"cron-expression"`         // cron执行表达式
}

// 日志配置
type Logs struct {
	Level      zapcore.Level `mapstructure:"level" json:"level"`
	Path       string        `mapstructure:"path" json:"path"`
	MaxSize    int           `mapstructure:"max-size" json:"maxSize"`
	MaxBackups int           `mapstructure:"max-backups" json:"maxBackups"`
	MaxAge     int           `mapstructure:"max-age" json:"maxAge"`
	Compress   bool          `mapstructure:"compress" json:"compress"`
}

// jwt配置
type JwtConfig struct {
	Realm      string `mapstructure:"realm" json:"realm"`
	Key        string `mapstructure:"key" json:"key"`
	Timeout    int    `mapstructure:"timeout" json:"timeout"`
	MaxRefresh int    `mapstructure:"max-refresh" json:"maxRefresh"`
}

// 数据库信息
type Database struct {
	Driver string `mapstructure:"driver" json:"driver"`
}

// MYSQL配置
type MysqlConfig struct {
	Username    string `mapstructure:"username" json:"username"`
	Password    string `mapstructure:"password" json:"password"`
	Database    string `mapstructure:"database" json:"database"`
	Host        string `mapstructure:"host" json:"host"`
	Port        int    `mapstructure:"port" json:"port"`
	Query       string `mapstructure:"query" json:"query"`
	LogMode     bool   `mapstructure:"log-mode" json:"logMode"`
	TablePrefix string `mapstructure:"table-prefix" json:"tablePrefix"`
	Charset     string `mapstructure:"charset" json:"charset"`
	Collation   string `mapstructure:"collation" json:"collation"`
}

// Sqlite配置结构体
type SqliteConfig struct {
	Path    string `mapstructure:"path" json:"path"`         // SQLite数据库文件路径
	LogMode bool   `mapstructure:"log-model" json:"logMode"` // 是否开启调试日志
}

// redis配置
type RedisConfig struct {
	Host         string `mapstructure:"host" json:"host"`
	Port         int    `mapstructure:"port" json:"port"`
	Password     string `mapstructure:"password" json:"password"`
	DB           int    `mapstructure:"db" json:"db"`
	PoolSize     int    `mapstructure:"pool-size" json:"poolSize"`          // 连接池最大连接数
	MinIdleConns int    `mapstructure:"min-idle-conns" json:"minIdleConns"` // 连接池最小空闲连接数
	MaxRetries   int    `mapstructure:"max-retries" json:"maxRetries"`      // 最大重试次数
	// 超时设置
	DialTimeout  int `mapstructure:"dial-timeout" json:"dialTimeout"`   // 拨号超时时间(秒)
	ReadTimeout  int `mapstructure:"read-timeout" json:"readTimeout"`   // 读取超时时间(秒)
	WriteTimeout int `mapstructure:"write-timeout" json:"writeTimeout"` // 写入超时时间(秒)
	PoolTimeout  int `mapstructure:"pool-timeout" json:"poolTimeout"`   // 从连接池获取连接的超时时间(秒)
	// 空闲连接超时时间(秒)
	IdleTimeout int `mapstructure:"idle-timeout" json:"idleTimeout"`
}

type RateLimitConfig struct {
	FillInterval int64 `mapstructure:"fill-interval" json:"fillInterval"`
	Capacity     int64 `mapstructure:"capacity" json:"capacity"`
}

type OnnxConfig struct {
	UseCuda        bool                     `mapstructure:"use-cuda" json:"useCuda"`
	FFmpegResize   bool                     `mapstructure:"ffmpeg-resize" json:"ffmpegResize"`
	MaxConcurrency int                      `mapstructure:"max-concurrency" json:"maxConcurrency"`
	Confidence     float32                  `mapstructure:"confidence" json:"confidence"`
	NmsThreshold   float32                  `mapstructure:"nms-threshold" json:"nmsThreshold"` // NMS阈值
	Base           *OnnxBaseConfig          `mapstructure:"base" json:"base"`
	FrameCount     int                      `mapstructure:"frame-count" json:"frameCount"`
	DefaultModel   string                   `mapstructure:"default-model" json:"defaultModel"`
	Models         map[string]*DetectConfig `mapstructure:"models" json:"models"`
	ROIDetectMode  string                   `mapstructure:"roi-detect-mode" json:"roiDetectMode"`
	// 推理引擎类型 onnx / tensorrt
	InferEngine string `mapstructure:"infer-engine" json:"inferEngine"`
	// TensorRT gRPC 服务地址
	TensorRTGrpcAddr string `mapstructure:"tensorrt-grpc-addr" json:"tensorrtGrpcAddr"`
}

type OnnxBaseConfig struct {
	InputName  string `mapstructure:"input-name" json:"inputName"`   // 模型输入节点名
	OutputName string `mapstructure:"output-name" json:"outputName"` // 模型输出节点名
}

// 通用检测模型配置
type DetectConfig struct {
	ModelPath   string  `mapstructure:"model-path" json:"modelPath"`     // ONNX模型文件路径
	InputShape  []int64 `mapstructure:"input-shape" json:"inputShape"`   // 输入张量形状
	OutputShape []int64 `mapstructure:"output-shape" json:"outputShape"` // 输出张量形状
	Confidence  float32 `mapstructure:"confidence" json:"confidence"`
}

// pcl配置
type PlcConfig struct {
	IP   string `mapstructure:"ip" json:"ip"`
	Rack int    `mapstructure:"rack" json:"rack"`
	Slot int    `mapstructure:"slot" json:"slot"`
}

// 读取配置信息
func InitConfig() {
	workDir, err := os.Getwd() // 获取目录
	if err != nil {
		panic(fmt.Errorf("读取应用目录失败:%s", err))
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/")
	// 读取配置信息
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败:%s", err))
	}

	// 解析到配置结构体
	if err := viper.Unmarshal(Conf); err != nil {
		panic(fmt.Errorf("初始化配置文件失败:%s", err))
	}

	// 开启热更新监听
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("配置文件已修改: %s", e.Name)
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Println("热更新配置失败: %v", err) // 不崩溃，只记录错误
		}
	})

	// 部分配合通过环境变量加载
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver != "" {
		Conf.Database.Driver = dbDriver
	}
	mysqlHost := os.Getenv("MYSQL_HOST")
	if mysqlHost != "" {
		Conf.Mysql.Host = mysqlHost
	}
	mysqlUsername := os.Getenv("MYSQL_USERNAME")
	if mysqlUsername != "" {
		Conf.Mysql.Username = mysqlUsername
	}
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	if mysqlPassword != "" {
		Conf.Mysql.Password = mysqlPassword
	}
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")
	if mysqlDatabase != "" {
		Conf.Mysql.Database = mysqlDatabase
	}
	mysqlPort := os.Getenv("MYSQL_PORT")
	if mysqlPort != "" {
		Conf.Mysql.Port, _ = strconv.Atoi(mysqlPort)
	}

	// 加载Redis相关环境变量
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost != "" {
		Conf.Redis.Host = redisHost
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort != "" {
		Conf.Redis.Port, _ = strconv.Atoi(redisPort)
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword != "" {
		Conf.Redis.Password = redisPassword
	}
	redisDB := os.Getenv("REDIS_DB")
	if redisDB != "" {
		Conf.Redis.DB, _ = strconv.Atoi(redisDB)
	}
	redisPoolSize := os.Getenv("REDIS_POOL_SIZE")
	if redisPoolSize != "" {
		Conf.Redis.PoolSize, _ = strconv.Atoi(redisPoolSize)
	}

	// 加载Sqlite环境变量
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath != "" {
		Conf.Sqlite.Path = sqlitePath
	}
	sqliteLogMode := os.Getenv("SQLITE_LOG_MODE")
	if sqliteLogMode != "" {
		Conf.Sqlite.LogMode, _ = strconv.ParseBool(sqliteLogMode)
	}
}

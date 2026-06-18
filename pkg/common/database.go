/*
*
@Time : 2025/12/14 18:40
@Author: FangYao( 方少、)
@Description:  初始化数据库
@Email: fy20030315@163.com
*/
package common

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-nvr/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	_ "modernc.org/sqlite"
	"time"
)

// 全局数据库对象
var DB *gorm.DB

// 全局Redis客户端对象
var Redis *redis.Client

func InitDB() {
	switch config.Conf.Database.Driver {
	case "mysql":
		DB = ConnMysql()
		//Redis = ConnRedis()
	case "sqlite3":
		DB = ConnSqlite()
		//Redis = ConnRedis()
	default:
		fmt.Printf("不支持的数据库驱动类型：%s", config.Conf.Database.Driver)
	}
}

// 连接mysql
func ConnMysql() *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&collation=%s&%s",
		config.Conf.Mysql.Username,
		config.Conf.Mysql.Password,
		config.Conf.Mysql.Host,
		config.Conf.Mysql.Port,
		config.Conf.Mysql.Database,
		config.Conf.Mysql.Charset,
		config.Conf.Mysql.Collation,
		config.Conf.Mysql.Query,
	)
	// 隐藏密码
	showDsn := fmt.Sprintf(
		"%s:******@tcp(%s:%d)/%s?charset=%s&collation=%s&%s",
		config.Conf.Mysql.Username,
		config.Conf.Mysql.Host,
		config.Conf.Mysql.Port,
		config.Conf.Mysql.Database,
		config.Conf.Mysql.Charset,
		config.Conf.Mysql.Collation,
		config.Conf.Mysql.Query,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 禁用外键
		DisableForeignKeyConstraintWhenMigrating: true,
		// 禁用表名复数形式
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
	})

	if err != nil {
		Log.Panic("初始化mysql数据库异常: %v", err)
	}
	// 开启mysql日志
	if config.Conf.Mysql.LogMode {
		db = db.Debug()
	}
	Log.Infof("初始化Mysql数据库完成! 连接信息: %s", showDsn)
	return db
}

// 连接redis
func ConnRedis() *redis.Client {
	// 构建Redis地址
	redisAddr := fmt.Sprintf("%s:%d", config.Conf.Redis.Host, config.Conf.Redis.Port)

	// 创建Redis客户端配置
	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     config.Conf.Redis.Password, // 密码
		DB:           config.Conf.Redis.DB,       // 数据库编号
		PoolSize:     config.Conf.Redis.PoolSize,
		MinIdleConns: config.Conf.Redis.MinIdleConns,
		MaxRetries:   config.Conf.Redis.MaxRetries,
		// 超时设置
		DialTimeout:  time.Duration(config.Conf.Redis.DialTimeout) * time.Second, // 先转Duration再相乘
		ReadTimeout:  time.Duration(config.Conf.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Conf.Redis.WriteTimeout) * time.Second,
		PoolTimeout:  time.Duration(config.Conf.Redis.PoolTimeout) * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		Log.Panic("Redis连接失败: 地址=%s, 数据库=%d, 错误=%v", redisAddr, config.Conf.Redis.DB, err)
	}

	// 连接信息展示
	var showAddr string
	if config.Conf.Redis.Password != "" {
		showAddr = fmt.Sprintf("%s:%d/%d (使用密码)", config.Conf.Redis.Host, config.Conf.Redis.Port, config.Conf.Redis.DB)
	} else {
		showAddr = fmt.Sprintf("%s:%d/%d (无密码)", config.Conf.Redis.Host, config.Conf.Redis.Port, config.Conf.Redis.DB)
	}

	Log.Infof("初始化Redis数据库完成 连接信息: %s", showAddr)
	return client
}

// 链接Sqlite
func ConnSqlite() *gorm.DB {
	sqlitePath := config.Conf.Sqlite.Path
	if sqlitePath == "" {
		Log.Panic("初始化sqlite数据库异常: 未配置sqlite文件路径")
	}

	// file:前缀是modernc.org/sqlite的标准格式，_foreign_keys=off禁用外键
	dsn := fmt.Sprintf("file:%s?_foreign_keys=off&cache=shared", sqlitePath)

	// 打开SQLite连接
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        dsn,
	}), &gorm.Config{
		// 禁用外键
		DisableForeignKeyConstraintWhenMigrating: true,
		// 禁用表名复数形式
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		Log.Panic("初始化sqlite数据库异常: 路径=%s, 错误=%v", sqlitePath, err)
	}

	// 调试日志
	if config.Conf.Mysql.LogMode {
		db = db.Debug()
	}

	// 配置底层连接池
	sqlDB, err := db.DB()
	if err != nil {
		Log.Panic("获取sqlite底层DB对象异常: %v", err)
	}
	sqlDB.SetMaxOpenConns(10)                  // 最大打开连接数
	sqlDB.SetMaxIdleConns(5)                   // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(0)                // 连接永不过期
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 空闲连接超时时间

	// 连接成功日志
	Log.Infof("初始化Sqlite数据库完成! 连接信息: 文件路径=%s", sqlitePath)

	return db
}

// 关闭Redis连接
func CloseRedis() error {
	if Redis != nil {
		return Redis.Close()
	}
	return nil
}

// Package db 提供数据库操作支持，支持读写分离
//
// 对应 SlightPHP 的 SDb 插件。
// 基于 database/sql 标准库，支持 MySQL、PostgreSQL、SQLite 等数据库。
// 支持一主多从的读写分离架构。
package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// 数据库配置
// ---------------------------------------------------------------------------

// Config 数据库配置
type Config struct {
	// Driver 数据库驱动名称: "mysql", "postgres", "sqlite3"
	Driver string `json:"driver"`

	// DSN 主库连接字符串（写库）
	DSN string `json:"dsn"`

	// Reads 从库连接字符串列表（读库），为空时读写都使用主库
	Reads []string `json:"reads,omitempty"`

	// MaxOpenConns 最大打开连接数
	MaxOpenConns int `json:"max_open_conns"`

	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int `json:"max_idle_conns"`

	// ConnMaxLifetime 连接最大存活时间
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// ConnMaxIdleTime 连接最大空闲时间
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// DefaultConfig 返回默认数据库配置
func DefaultConfig() *Config {
	return &Config{
		Driver:          "mysql",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// ---------------------------------------------------------------------------
// DB 主结构
// ---------------------------------------------------------------------------

// DB 封装了数据库读写分离连接池
type DB struct {
	config    *Config
	write     *sql.DB
	reads     []*sql.DB
	readIndex uint64
}

// New 根据配置创建一个新的数据库连接池
func New(cfg *Config) (*DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	db := &DB{
		config:    cfg,
		reads:     make([]*sql.DB, 0),
		readIndex: 0,
	}

	// 打开写库连接
	write, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: open write connection: %w", err)
	}
	db.write = write

	// 配置连接池
	applyPoolConfig(write, cfg)

	// 打开读库连接
	for _, dsn := range cfg.Reads {
		read, err := sql.Open(cfg.Driver, dsn)
		if err != nil {
			log.Printf("[DB] warning: open read connection failed: %v", err)
			continue
		}
		applyPoolConfig(read, cfg)
		db.reads = append(db.reads, read)
	}

	return db, nil
}

// NewFromDSN 从单个 DSN 创建数据库连接（无读写分离）
func NewFromDSN(driver, dsn string) (*DB, error) {
	cfg := DefaultConfig()
	cfg.Driver = driver
	cfg.DSN = dsn
	return New(cfg)
}

// applyPoolConfig 对 sql.DB 实例应用连接池配置
func applyPoolConfig(db *sql.DB, cfg *Config) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
}

// ---------------------------------------------------------------------------
// 连接获取
// ---------------------------------------------------------------------------

// W 返回写库连接
func (db *DB) W() *sql.DB {
	return db.write
}

// R 返回读库连接（使用轮询负载均衡）
// 如果没有配置从库，则返回写库连接
func (db *DB) R() *sql.DB {
	if len(db.reads) == 0 {
		return db.write
	}
	if len(db.reads) == 1 {
		return db.reads[0]
	}
	// 原子轮询
	idx := atomic.AddUint64(&db.readIndex, 1) % uint64(len(db.reads))
	return db.reads[idx]
}

// ---------------------------------------------------------------------------
// 查询方法
// ---------------------------------------------------------------------------

// Query 执行查询，自动路由到从库（若可用）
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.R().Query(query, args...)
}

// QueryRow 查询单行，自动路由到从库（若可用）
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.R().QueryRow(query, args...)
}

// Exec 执行写操作，路由到主库
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.W().Exec(query, args...)
}

// Prepare 准备查询，根据 isRead 决定使用读库还是写库
func (db *DB) Prepare(query string, isRead ...bool) (*sql.Stmt, error) {
	if len(isRead) > 0 && isRead[0] {
		return db.R().Prepare(query)
	}
	return db.W().Prepare(query)
}

// ---------------------------------------------------------------------------
// 事务
// ---------------------------------------------------------------------------

// Begin 开始一个事务（在主库上）
func (db *DB) Begin() (*sql.Tx, error) {
	return db.W().Begin()
}

// Transaction 执行一个事务
// 函数返回 error 时事务回滚，否则提交
func (db *DB) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.W().Begin()
	if err != nil {
		return fmt.Errorf("db: begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("db: rollback failed: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// 连接管理
// ---------------------------------------------------------------------------

// Ping 检查数据库连接
func (db *DB) Ping() error {
	if err := db.write.Ping(); err != nil {
		return fmt.Errorf("db: ping write: %w", err)
	}
	for i, read := range db.reads {
		if err := read.Ping(); err != nil {
			return fmt.Errorf("db: ping read[%d]: %w", i, err)
		}
	}
	return nil
}

// Close 关闭所有数据库连接
func (db *DB) Close() error {
	if err := db.write.Close(); err != nil {
		return fmt.Errorf("db: close write: %w", err)
	}
	for i, read := range db.reads {
		if err := read.Close(); err != nil {
			return fmt.Errorf("db: close read[%d]: %w", i, err)
		}
	}
	return nil
}

// Stats 返回数据库统计信息
func (db *DB) Stats() *sql.DBStats {
	stats := db.write.Stats()
	return &stats
}

// ---------------------------------------------------------------------------
// 表模型辅助
// ---------------------------------------------------------------------------

// Model 提供便捷的表操作接口
type Model struct {
	db    *DB
	table string
}

// NewModel 创建一个表模型
func NewModel(database *DB, tableName string) *Model {
	return &Model{
		db:    database,
		table: tableName,
	}
}

// Table 获取或设置表名
func (m *Model) Table(name string) *Model {
	m.table = name
	return m
}

// FindByID 根据主键查找记录
func (m *Model) FindByID(id int64) (*sql.Row, error) {
	// 默认主键名为 id
	return m.db.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id = ?", m.table), id), nil
}

// Count 获取记录数
func (m *Model) Count(where string, args ...interface{}) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.table)
	if where != "" {
		query += " WHERE " + where
	}
	var count int
	err := m.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

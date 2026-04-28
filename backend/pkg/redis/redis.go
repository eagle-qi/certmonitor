package redis

import (
	"context"
	"fmt"

	"certmonitor/internal/config"
	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// NewClient 创建 Redis 客户端连接
func NewClient(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	Client = rdb
	return rdb, nil
}

// Set 设置缓存值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Client.Set(ctx, key, value, expiration).Err()
}

// Get 获取缓存值
func Get(ctx context.Context, key string) (string, error) {
	return Client.Get(ctx, key).Result()
}

// Del 删除缓存键
func Del(ctx context.Context, keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return Client.Exists(ctx, keys...).Result()
}

// SetJSON 以 JSON 格式设置值
func SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Set(ctx, key, data, expiration)
}

// GetJSON 以 JSON 格式获取值并反序列化
func GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// Incr 自增计数器
func Incr(ctx context.Context, key string) (int64, error) {
	return Client.Incr(ctx, key).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

// HSet Hash 结构设置字段
func HSet(ctx context.Context, key string, values ...interface{}) error {
	return Client.HSet(ctx, key, values...).Err()
}

// HGet Hash 结构获取字段值
func HGet(ctx context.Context, key, field string) (string, error) {
	return Client.HGet(ctx, key, field).Result()
}

// ===========================================
// 验证码相关缓存操作
// ===========================================

const (
	CaptchaPrefix     = "certmonitor:captcha:"
	CaptchaExpireTime = 10 * time.Minute // 验证码有效期10分钟
	CaptchaMaxCount   = 5                // 同一邮箱每天最大发送次数
	CaptchaDayLimit   = "certmonitor:captcha_count:"
)

// SaveCaptcha 保存验证码到 Redis
func SaveCaptcha(ctx context.Context, email, captcha, captchaType string) error {
	key := CaptchaPrefix + email + ":" + captchaType

	// 存储验证码
	err := Set(ctx, key, captcha, CaptchaExpireTime)
	if err != nil {
		return err
	}

	// 计数器：限制每日发送次数
	todayKey := CaptchaDayLimit + email + ":" + time.Now().Format("2006-01-02")
	count, _ := Incr(ctx, todayKey)
	if count == 1 {
		// 首次发送，设置24小时过期
		Expire(ctx, todayKey, 24*time.Hour)
	}

	return nil
}

// VerifyCaptcha 校验验证码
func VerifyCaptcha(ctx context.Context, email, captcha, captchaType string) (bool, error) {
	key := CaptchaPrefix + email + ":" + captchaType

	stored, err := Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if stored == captcha {
		Del(ctx, key) // 验证后立即删除
		return true, nil
	}

	return false, nil
}

// CheckCaptchaRateLimit 检查验证码发送频率限制
func CheckCaptchaRateLimit(ctx context.Context, email string) (bool, error) {
	todayKey := CaptchaDayLimit + email + ":" + time.Now().Format("2006-01-02")

	count, err := Get(ctx, todayKey)
	if err != nil && err != redis.Nil {
		return false, err
	}

	currentCount := 0
	if count != "" {
		fmt.Sscanf(count, "%d", &currentCount)
	}

	return currentCount < CaptchaMaxCount, nil
}

// ===========================================
// 用户会话相关缓存操作
// ===========================================

const (
	SessionPrefix    = "certmonitor:session:"
	SessionExpire    = 2 * time.Hour
	LoginFailPrefix  = "certmonitor:login_fail:"
	LoginMaxAttempts = 5
	LockDuration     = 30 * time.Minute
)

// SaveUserSession 保存用户会话信息
func SaveUserSession(ctx context.Context, userID uint64, userInfo map[string]interface{}) error {
	key := SessionPrefix + fmt.Sprintf("%d", userID)
	return SetJSON(ctx, key, userInfo, SessionExpire)
}

// GetUserSession 获取用户会话信息
func GetUserSession(ctx context.Context, userID uint64) (map[string]interface{}, error) {
	key := SessionPrefix + fmt.Sprintf("%d", userID)
	var result map[string]interface{}
	err := GetJSON(ctx, key, &result)
	return result, err
}

// DeleteUserSession 删除用户会话（强制下线）
func DeleteUserSession(ctx context.Context, userID uint64) error {
	key := SessionPrefix + fmt.Sprintf("%d", userID)
	return Del(ctx, key)
}

// RecordLoginFailure 记录登录失败
func RecordLoginFailure(ctx context.Context, usernameOrEmail string) (int, error) {
	key := LoginFailPrefix + usernameOrEmail

	count, err := Incr(ctx, key)
	if err != nil {
		return 0, err
	}

	if count == 1 {
		Expire(ctx, key, LockDuration)
	}

	return int(count), nil
}

// ClearLoginFailures 清除登录失败记录（登录成功后）
func ClearLoginFailures(ctx context.Context, usernameOrEmail string) error {
	key := LoginFailPrefix + usernameOrEmail
	return Del(ctx, key)
}

// IsAccountLocked 检查账号是否被锁定
func IsAccountLocked(ctx context.Context, usernameOrEmail string) (bool, error) {
	count, err := Get(ctx, LoginFailPrefix+usernameOrEmail)
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	attempts := 0
	fmt.Sscanf(count, "%d", &attempts)
	return attempts >= LoginMaxAttempts, nil
}

// ===========================================
// 探测任务状态缓存操作
// ===========================================

const TaskStatusPrefix = "certmonitor:task:status:"

// SetTaskStatus 缓存探测任务状态
func SetTaskStatus(ctx context.Context, taskID uint64, status map[string]interface{}) error {
	key := TaskStatusPrefix + fmt.Sprintf("%d", taskID)
	return SetJSON(ctx, key, status, 24*time.Hour)
}

// GetTaskStatus 获取探测任务状态
func GetTaskStatus(ctx context.Context, taskID uint64) (map[string]interface{}, error) {
	key := TaskStatusPrefix + fmt.Sprintf("%d", taskID)
	var result map[string]interface{}
	err := GetJSON(ctx, key, &result)
	return result, err
}

package config

import (
	"log"
	"os"
	"strings"
	"time"
)

//
// ==============================
// 🧠 配置结构体定义
// ==============================
//

// DiagnosisConfig 表示诊断系统相关配置项
type DiagnosisConfig struct {
	CleanInterval            time.Duration // 清理器执行间隔
	WriteInterval            time.Duration // 日志写入间隔
	RetentionRawDuration     time.Duration // 原始事件保留时间
	RetentionCleanedDuration time.Duration // 清理池保留时间
	UnreadyThresholdDuration time.Duration //告警与邮件发送时间间隔
	AlertDispatchInterval    time.Duration // 邮件轮询检测发送间隔（独立于异常阈值）
}

// KubernetesConfig 表示 Kubernetes API 健康检查相关配置项
type KubernetesConfig struct {
	APIHealthCheckInterval time.Duration // /healthz 探测间隔
}

// MailerConfig 表示邮件发送相关配置项
type MailerConfig struct {
	SMTPHost string   // 邮件服务器地址
	SMTPPort string   // 邮件服务器端口
	Username string   // 登录账号
	Password string   // 登录密码或授权码
	From     string   // 发件人邮箱
	To       []string // 收件人列表（支持多个）
}

// AppConfig 是整个系统的顶层配置结构体
type AppConfig struct {
	Diagnosis  DiagnosisConfig
	Kubernetes KubernetesConfig
	Mailer     MailerConfig
}

// GlobalConfig 是对外暴露的全局配置实例
var GlobalConfig AppConfig

//
// ==============================
// 🔧 默认值定义
// ==============================
//

// 默认时间配置（支持覆盖）
var defaultDurations = map[string]string{
	"DIAGNOSIS_CLEAN_INTERVAL":             "30s",
	"DIAGNOSIS_WRITE_INTERVAL":             "30s",
	"DIAGNOSIS_RETENTION_RAW_DURATION":     "10m",
	"DIAGNOSIS_RETENTION_CLEANED_DURATION": "5m",
	"KUBERNETES_API_HEALTH_CHECK_INTERVAL": "15s",
	"DIAGNOSIS_UNREADY_THRESHOLD_DURATION": "30s",
	"DIAGNOSIS_ALERT_DISPATCH_INTERVAL":    "30s",
}

// 默认字符串配置（支持覆盖）
var defaultStrings = map[string]string{
	"MAIL_SMTP_HOST": "smtp.gmail.com",
	"MAIL_SMTP_PORT": "587",
}

//
// ==============================
// 🧩 配置加载入口
// ==============================
//

// LoadConfig 加载所有配置项（支持 ENV 覆盖）
func LoadConfig() {
	log.Println("🔧 加载配置中 ...")

	GlobalConfig.Diagnosis = DiagnosisConfig{
		CleanInterval:            getDuration("DIAGNOSIS_CLEAN_INTERVAL"),
		WriteInterval:            getDuration("DIAGNOSIS_WRITE_INTERVAL"),
		RetentionRawDuration:     getDuration("DIAGNOSIS_RETENTION_RAW_DURATION"),
		RetentionCleanedDuration: getDuration("DIAGNOSIS_RETENTION_CLEANED_DURATION"),
		UnreadyThresholdDuration: getDuration("DIAGNOSIS_UNREADY_THRESHOLD_DURATION"),
		AlertDispatchInterval:    getDuration("DIAGNOSIS_ALERT_DISPATCH_INTERVAL"),
	}

	GlobalConfig.Kubernetes = KubernetesConfig{
		APIHealthCheckInterval: getDuration("KUBERNETES_API_HEALTH_CHECK_INTERVAL"),
	}

	GlobalConfig.Mailer = MailerConfig{
		SMTPHost: getString("MAIL_SMTP_HOST"),
		SMTPPort: getString("MAIL_SMTP_PORT"),
		Username: getString("MAIL_USERNAME"),
		Password: getString("MAIL_PASSWORD"),
		From:     getString("MAIL_FROM"),
		To:       getStringList("MAIL_TO"),
	}

	log.Printf("✅ 配置加载完成: %+v\n", GlobalConfig)
}

//
// ==============================
// 🧪 工具函数（ENV 优先，默认值兜底）
// ==============================
//

// getDuration 获取时间配置（如 30s、5m）
func getDuration(envKey string) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		log.Printf("⚠️ 环境变量 %s 格式错误（期望如 10s/5m），将使用默认值", envKey)
	}
	def, ok := defaultDurations[envKey]
	if !ok {
		log.Fatalf("❌ 未定义默认时间配置项: %s", envKey)
	}
	d, err := time.ParseDuration(def)
	if err != nil {
		log.Fatalf("❌ 默认时间配置项格式错误: %s = %s", envKey, def)
	}
	return d
}

// getString 获取字符串配置
func getString(envKey string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	def, ok := defaultStrings[envKey]
	if !ok {
		log.Fatalf("❌ 未定义默认字符串配置项: %s", envKey)
	}
	log.Printf("⚠️ 环境变量 %s 未设置，使用默认值", envKey)
	return def
}

// getStringList 获取字符串列表配置（使用逗号分隔）
func getStringList(envKey string) []string {
	if val := os.Getenv(envKey); val != "" {
		list := strings.Split(val, ",")
		for i := range list {
			list[i] = strings.TrimSpace(list[i])
		}
		return list
	}
	def, ok := defaultStrings[envKey]
	if !ok {
		log.Fatalf("❌ 未定义默认列表配置项: %s", envKey)
	}
	log.Printf("⚠️ 环境变量 %s 未设置，使用默认收件人列表", envKey)
	return strings.Split(def, ",")
}

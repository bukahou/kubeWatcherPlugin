// =======================================================================================
// 📄 diagnosis/diagnosis_init.go
//
// ✨ Description:
//     Entry point for starting the diagnosis system.
//     Initializes and launches both the log cleaner and log writer.
//
// 📦 Responsibilities:
//     - Configure intervals for cleaning and writing logs
//     - Start the cleaner loop (deduplication + retention)
//     - Start the file writer loop (deduplicated persistent logs)
// =======================================================================================

package bootstrap

import (
	"NeuroController/config"
	"NeuroController/internal/diagnosis"
	"fmt"
	"time"
)

// ✅ 启动诊断系统：包括清理器和日志写入器
func StartDiagnosisSystem() {

	// ✅ 从配置中获取
	cleanInterval := config.GlobalConfig.Diagnosis.CleanInterval
	writeInterval := config.GlobalConfig.Diagnosis.WriteInterval

	// ✅ 启动提示
	fmt.Println("🧠 正在启动诊断系统 ...")
	fmt.Printf("🧼 清理间隔：%v\n", cleanInterval)
	fmt.Printf("📝 写入间隔：%v\n", writeInterval)

	// 启动清理器（执行去重和过期清理）
	diagnosis.StartCleanerLoop(cleanInterval)

	// 启动日志写入器（定期将去重后的日志写入文件）
	go func() {
		for {
			diagnosis.WriteNewCleanedEventsToFile()
			time.Sleep(writeInterval)
		}
	}()

	fmt.Println("✅ 诊断系统启动成功。")
}

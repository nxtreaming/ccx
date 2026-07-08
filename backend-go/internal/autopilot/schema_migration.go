package autopilot

import (
	"database/sql"
	"fmt"
	"log"
)

// ── Autopilot SQLite schema 版本化（设计 §12.2 P1.5）──
//
// 7 个 autopilot store（ProfileStore/ProfileChangelogStore/LocalRuntimeStore/
// SubscriptionStore/TraceStore/ManualIntentStore/AdvisorDecisionStore）全部
// 复用同一个 *sql.DB 连接（由 ProfileStore 打开，其余 store 通过 store.DB() 接力）。
// 因此 schema 版本是这个 DB 文件级别的单一版本，不是每张表各自一个版本。
//
// 参考 internal/metrics/sqlite_store.go 的 PRAGMA user_version 迁移模式。

// autopilotSchemaVersion 当前代码期望的 schema 版本。
// v1 = 现有 7 张表的建表语句本身（首次引入版本化时的基线，无需 ALTER）。
// 后续新增/变更表结构时，在 ensureSchemaVersion 里追加 "if version < N { ... }" 迁移块，
// 并将本常量递增。
const autopilotSchemaVersion = 1

// ensureSchemaVersion 在任何 CREATE TABLE 之前执行一次版本检查/迁移。
// 必须在 ProfileStore 打开 DB 后、调用 initProfileStoreSchema 之前调用——
// 它是唯一的 DB 打开入口，其余 6 个 store 复用同一连接，不需要重复调用。
//
//   - version == 0（全新库）：直接写 PRAGMA user_version = autopilotSchemaVersion，
//     交给后续各 initXxxSchema 的 CREATE TABLE IF NOT EXISTS 建表。
//   - version == autopilotSchemaVersion：无需操作。
//   - 0 < version < autopilotSchemaVersion：未来加 v2+ 时，在此插入门控迁移块。
//   - version > autopilotSchemaVersion：库版本比当前代码新（例如降级部署），
//     返回 error，不做任何写操作——调用方（NewProfileStore）会把这个 error
//     原样传播到 main.go 的现有 fail-open 分支（禁用 Autopilot，保留现有调度）。
func ensureSchemaVersion(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("[Autopilot-SchemaMigration] db 为 nil")
	}

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("[Autopilot-SchemaMigration] 读取 PRAGMA user_version 失败: %w", err)
	}

	if version > autopilotSchemaVersion {
		return fmt.Errorf(
			"[Autopilot-SchemaMigration] 数据库 schema 版本 (%d) 高于当前代码支持的版本 (%d)，"+
				"可能是降级部署或数据库损坏，拒绝启动 Autopilot 以避免数据不一致",
			version, autopilotSchemaVersion,
		)
	}

	if version == autopilotSchemaVersion {
		return nil
	}

	// 未来 v2+ 的门控迁移块示例（保留注释，实际迁移到来时按此形状追加）：
	//
	// if version < 2 {
	//     migrations := []string{
	//         "ALTER TABLE autopilot_endpoint_profiles ADD COLUMN xxx TEXT DEFAULT ''",
	//         "PRAGMA user_version = 2",
	//     }
	//     for _, stmt := range migrations {
	//         if _, err := db.Exec(stmt); err != nil {
	//             return fmt.Errorf("[Autopilot-SchemaMigration] v1->v2 迁移失败: %w", err)
	//         }
	//     }
	//     log.Printf("[Autopilot-SchemaMigration] schema 升级: v1 -> v2")
	//     version = 2
	// }

	// version == 0：全新库，直接写入当前基线版本；version 属于 (0, autopilotSchemaVersion) 但
	// 未命中任何迁移块的情况理论上不应出现（说明版本常量与迁移块不同步），同样兜底写回当前版本，
	// 不阻塞启动——迁移块本身的正确性由后续新增迁移时的测试覆盖。
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", autopilotSchemaVersion)); err != nil {
		return fmt.Errorf("[Autopilot-SchemaMigration] 写入 PRAGMA user_version 失败: %w", err)
	}

	if version != 0 {
		log.Printf("[Autopilot-SchemaMigration] schema 版本已从 %d 更新至 %d", version, autopilotSchemaVersion)
	}

	return nil
}

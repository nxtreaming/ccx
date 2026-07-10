// Package presetstore 提供可远程增量更新的"预置数据"运行时存储。
//
// 预置数据（订阅来源分类、模型注册表、渠道预置、内置模型清单）原先散落在
// 编译期常量里，每次微调都要发版。presetstore 把它们抽成统一的 PresetBundle，
// 支持三层优先级：编译期内置（embedded）< 磁盘缓存（cache）< 远程覆盖（remote）。
//
// 本文件只定义数据契约与订阅来源预置；远程拉取/校验/缓存在 Phase 2 补充。
package presetstore

// CurrentSchemaVersion 是本二进制支持的 subscription-preset schema 主版本。
// 远程 schemaVersion 高于此值时整体弃用远程（老客户端不误解新结构）。
const CurrentSchemaVersion = 1

// PresetBundle 是所有预置数据类的聚合，运行时由 PresetStore 原子持有。
type PresetBundle struct {
	// SchemaVersion 结构版本；仅在字段不兼容变更时递增。
	SchemaVersion int `json:"schemaVersion"`
	// DataVersion 数据版本，单调递增字符串序比较（如 "2026.07.10-1"）。
	// embedded 默认为空串，任何非空远程版本都视为更新。
	DataVersion string `json:"dataVersion,omitempty"`

	// Subscription 订阅来源预置（本期唯一新增数据类）。
	Subscription SubscriptionPreset `json:"subscription"`
}

// SubscriptionPreset 是订阅中心的来源分类预置。
type SubscriptionPreset struct {
	// OriginTypes 来源类型及其推导出的信任等级。
	OriginTypes []OriginTypeEntry `json:"originTypes"`
	// BillingModes 计费模式枚举。
	BillingModes []string `json:"billingModes"`
	// Sources 订阅来源枚举（手动/自动发现）。
	Sources []string `json:"sources"`
	// AutoRefreshProviders 支持自动余额刷新的 provider 白名单。
	AutoRefreshProviders []string `json:"autoRefreshProviders"`
	// NewApiDefaults new-api 家族站点接入时的建议预填值。
	NewApiDefaults NewApiDefaults `json:"newApiDefaults"`
	// OriginTypeAliases 历史枚举归一化映射（如 public_benefit -> community）。
	// 读取存量数据时用于兼容，键为旧值、值为规范值。
	OriginTypeAliases map[string]string `json:"originTypeAliases,omitempty"`
}

// OriginTypeEntry 是单个来源类型到信任等级的映射。
type OriginTypeEntry struct {
	Value string `json:"value"`
	Tier  string `json:"tier"`
}

// NewApiDefaults 是 new-api 接入的建议预填值。
type NewApiDefaults struct {
	OriginType  string `json:"originType"`
	OriginTier  string `json:"originTier"`
	BillingMode string `json:"billingMode"`
}

// TierFor 返回给定来源类型（先经别名归一化）推导出的信任等级；
// 未命中返回 "unknown"。
func (s SubscriptionPreset) TierFor(originType string) string {
	canonical := s.Canonicalize(originType)
	for _, e := range s.OriginTypes {
		if e.Value == canonical {
			return e.Tier
		}
	}
	return "unknown"
}

// Canonicalize 把历史/别名来源类型归一化为规范值；无别名时原样返回。
func (s SubscriptionPreset) Canonicalize(originType string) string {
	if canonical, ok := s.OriginTypeAliases[originType]; ok {
		return canonical
	}
	return originType
}

// SupportsAutoRefresh 判断 provider 是否在自动余额刷新白名单内。
func (s SubscriptionPreset) SupportsAutoRefresh(provider string) bool {
	for _, p := range s.AutoRefreshProviders {
		if p == provider {
			return true
		}
	}
	return false
}

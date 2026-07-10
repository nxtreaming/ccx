package presetstore

import "fmt"

// knownTiers 是允许的信任等级白名单。
// 远程数据引入未知 tier 时该来源类型条目视为非法，整份数据弃用。
var knownTiers = map[string]bool{
	"first":   true,
	"second":  true,
	"third":   true,
	"local":   true,
	"unknown": true,
}

// Validate 校验一份候选 bundle 是否可安全采用。
//
// 预置数据会影响路由信任等级（tier），故按不可信输入严格校验：
// schema 兼容 + 枚举非空 + tier 白名单 + new-api 默认值自洽。
// 任一不满足返回 error，调用方应弃用该候选、保留当前生效数据。
func Validate(b *PresetBundle) error {
	if b == nil {
		return fmt.Errorf("[presetstore] bundle 为 nil")
	}
	if b.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("[presetstore] schemaVersion %d 高于本二进制支持的 %d，需升级版本",
			b.SchemaVersion, CurrentSchemaVersion)
	}

	sub := b.Subscription
	if len(sub.OriginTypes) == 0 {
		return fmt.Errorf("[presetstore] originTypes 不能为空")
	}

	seen := make(map[string]bool, len(sub.OriginTypes))
	for _, e := range sub.OriginTypes {
		if e.Value == "" {
			return fmt.Errorf("[presetstore] originType.value 不能为空")
		}
		if seen[e.Value] {
			return fmt.Errorf("[presetstore] originType %q 重复", e.Value)
		}
		seen[e.Value] = true
		if !knownTiers[e.Tier] {
			return fmt.Errorf("[presetstore] originType %q 的 tier %q 不在白名单内", e.Value, e.Tier)
		}
	}

	if len(sub.BillingModes) == 0 {
		return fmt.Errorf("[presetstore] billingModes 不能为空")
	}
	if len(sub.Sources) == 0 {
		return fmt.Errorf("[presetstore] sources 不能为空")
	}

	// new-api 建议值必须引用已知来源类型（经别名归一化后）。
	if d := sub.NewApiDefaults; d.OriginType != "" {
		if !seen[sub.Canonicalize(d.OriginType)] {
			return fmt.Errorf("[presetstore] newApiDefaults.originType %q 不是已知来源类型", d.OriginType)
		}
	}

	// 别名目标必须是已知规范值。
	for alias, canonical := range sub.OriginTypeAliases {
		if !seen[canonical] {
			return fmt.Errorf("[presetstore] originTypeAlias %q -> %q 的目标不是已知来源类型", alias, canonical)
		}
	}

	return nil
}

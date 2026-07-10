package presetstore

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// embedded/subscription-preset.json 是编译期内置的订阅来源预置默认值。
//
// 该文件必须与 shared/subscription-preset/subscription-preset.json 保持一致，
// 由 scripts/generate-preset-manifest.mjs 同步（go:embed 无法引用父目录，故需副本）。
// 它是永久 fallback：磁盘缓存/远程覆盖均不可用时使用，保证离线与首启可用。
//
//go:embed embedded/subscription-preset.json
var embeddedSubscriptionPreset []byte

// EmbeddedBundle 解析并返回编译期内置的 PresetBundle。
// 内置数据的 DataVersion 为空串，任何有效远程版本都会覆盖它。
// 解析失败视为构建缺陷（内置数据由本仓库控制），直接 panic。
func EmbeddedBundle() *PresetBundle {
	var sub SubscriptionPreset
	if err := json.Unmarshal(embeddedSubscriptionPreset, &sub); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 subscription-preset.json 解析失败: %v", err))
	}
	return &PresetBundle{
		SchemaVersion: CurrentSchemaVersion,
		DataVersion:   "", // 内置默认无数据版本
		Subscription:  sub,
	}
}

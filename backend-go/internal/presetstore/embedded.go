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

//go:embed embedded/model-registry.json
var embeddedModelRegistry []byte

//go:embed embedded/channel-presets.json
var embeddedChannelPresets []byte

//go:embed embedded/builtin-manifest.json
var embeddedBuiltinManifest []byte

//go:embed embedded/index.json
var embeddedIndex []byte

// EmbeddedBundle 解析并返回编译期内置的 PresetBundle。
// 内置数据携带生成时的数据版本，防止旧磁盘缓存覆盖更新后的二进制内置预置。
// 解析失败视为构建缺陷（内置数据由本仓库控制），直接 panic。
func EmbeddedBundle() *PresetBundle {
	var index PresetIndex
	if err := json.Unmarshal(embeddedIndex, &index); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 index.json 解析失败: %v", err))
	}
	if index.DataVersion == "" {
		panic("[presetstore] 内置 index.json 缺少 dataVersion")
	}

	var sub SubscriptionPreset
	if err := json.Unmarshal(embeddedSubscriptionPreset, &sub); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 subscription-preset.json 解析失败: %v", err))
	}

	var registry ModelRegistryPreset
	if err := json.Unmarshal(embeddedModelRegistry, &registry); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 model-registry.json 解析失败: %v", err))
	}

	var channelPresets ChannelPresetsPreset
	if err := json.Unmarshal(embeddedChannelPresets, &channelPresets); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 channel-presets.json 解析失败: %v", err))
	}

	var builtinManifest BuiltinModelsManifestPreset
	if err := json.Unmarshal(embeddedBuiltinManifest, &builtinManifest); err != nil {
		panic(fmt.Sprintf("[presetstore] 内置 builtin-manifest.json 解析失败: %v", err))
	}

	return &PresetBundle{
		SchemaVersion:          CurrentSchemaVersion,
		DataVersion:            index.DataVersion,
		Subscription:           sub,
		ModelRegistry:          &registry,
		ChannelPresets:         &channelPresets,
		BuiltinModelsManifests: &builtinManifest,
	}
}

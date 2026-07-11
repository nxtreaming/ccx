package presetstore

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

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

	if d := sub.NewApiDefaults; d.OriginType != "" {
		if !seen[sub.Canonicalize(d.OriginType)] {
			return fmt.Errorf("[presetstore] newApiDefaults.originType %q 不是已知来源类型", d.OriginType)
		}
	}

	for alias, canonical := range sub.OriginTypeAliases {
		if !seen[canonical] {
			return fmt.Errorf("[presetstore] originTypeAlias %q -> %q 的目标不是已知来源类型", alias, canonical)
		}
	}

	if b.ModelRegistry != nil {
		if err := validateModelRegistryPreset(b.ModelRegistry); err != nil {
			return err
		}
	}
	if b.ChannelPresets != nil {
		if err := validateChannelPresets(b.ChannelPresets); err != nil {
			return err
		}
	}
	if b.BuiltinModelsManifests != nil {
		if err := validateBuiltinModelsManifestPreset(b.BuiltinModelsManifests); err != nil {
			return err
		}
	}

	return nil
}

func validateModelRegistryPreset(preset *ModelRegistryPreset) error {
	for idx, capability := range preset.UpstreamCapabilities {
		if len(capability.Patterns) == 0 {
			return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].patterns 不能为空", idx)
		}
		for patternIdx, pattern := range capability.Patterns {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].patterns[%d] 不能为空", idx, patternIdx)
			}
		}
		if err := validateNonNegativeInt("contextWindowTokens", capability.ContextWindowTokens); err != nil {
			return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].%w", idx, err)
		}
		if err := validateNonNegativeInt("maxOutputTokens", capability.MaxOutputTokens); err != nil {
			return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].%w", idx, err)
		}
		if err := validateNonNegativeInt("defaultOutputTokens", capability.DefaultOutputTokens); err != nil {
			return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].%w", idx, err)
		}
		if err := validateNonNegativeInt("recommendedOutputTokens", capability.RecommendedOutputTokens); err != nil {
			return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].%w", idx, err)
		}
		if capability.Pricing != nil {
			if err := validatePricePointer("inputCacheHitPrice", capability.Pricing.InputCacheHitPrice); err != nil {
				return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.%w", idx, err)
			}
			if err := validatePricePointer("inputCacheMissPrice", capability.Pricing.InputCacheMissPrice); err != nil {
				return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.%w", idx, err)
			}
			if err := validatePricePointer("outputPrice", capability.Pricing.OutputPrice); err != nil {
				return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.%w", idx, err)
			}
			for tierIdx, tier := range capability.Pricing.Tiers {
				if err := validateNonNegativeInt("inputTokensAbove", tier.InputTokensAbove); err != nil {
					return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.tiers[%d].%w", idx, tierIdx, err)
				}
				if err := validateNonNegativeInt("inputTokensUpTo", tier.InputTokensUpTo); err != nil {
					return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.tiers[%d].%w", idx, tierIdx, err)
				}
				if err := validatePricePointer("inputCacheHitPrice", tier.InputCacheHitPrice); err != nil {
					return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.tiers[%d].%w", idx, tierIdx, err)
				}
				if err := validatePricePointer("inputCacheMissPrice", tier.InputCacheMissPrice); err != nil {
					return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.tiers[%d].%w", idx, tierIdx, err)
				}
				if err := validatePricePointer("outputPrice", tier.OutputPrice); err != nil {
					return fmt.Errorf("[presetstore] modelRegistry.upstreamCapabilities[%d].pricing.tiers[%d].%w", idx, tierIdx, err)
				}
			}
		}
	}
	return nil
}

func validateChannelPresets(preset *ChannelPresetsPreset) error {
	if preset.SchemaVersion != 1 {
		return fmt.Errorf("[presetstore] channelPresets.schemaVersion=%d，当前仅支持 1", preset.SchemaVersion)
	}
	required := []string{"claudeMessages", "openAIChat", "codexResponses", "openAIMessages"}
	for _, key := range required {
		raw, ok := preset.Collections[key]
		if !ok || len(raw) == 0 {
			return fmt.Errorf("[presetstore] channelPresets.%s 缺失", key)
		}
		var collection struct {
			SchemaVersion int `json:"schemaVersion"`
		}
		if err := json.Unmarshal(raw, &collection); err != nil {
			return fmt.Errorf("[presetstore] channelPresets.%s 解析失败: %w", key, err)
		}
		if collection.SchemaVersion != 1 {
			return fmt.Errorf("[presetstore] channelPresets.%s.schemaVersion=%d，当前仅支持 1", key, collection.SchemaVersion)
		}
	}
	return nil
}

func validateBuiltinModelsManifestPreset(preset *BuiltinModelsManifestPreset) error {
	seenServiceTypes := map[string]bool{
		"messages":  true,
		"openai":    true,
		"responses": true,
		"chat":      true,
		"gemini":    true,
		"images":    true,
		"vectors":   true,
	}
	for idx, manifest := range preset.Manifests {
		if strings.TrimSpace(manifest.BaseURLPattern) == "" {
			return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].baseUrlPattern 不能为空", idx)
		}
		if strings.Contains(manifest.BaseURLPattern, "://") {
			return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].baseUrlPattern 不能包含 scheme", idx)
		}
		if _, err := url.Parse("https://" + manifest.BaseURLPattern); err != nil {
			return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].baseUrlPattern 非法: %w", idx, err)
		}
		if !seenServiceTypes[manifest.ServiceType] {
			return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].serviceType %q 不在白名单内", idx, manifest.ServiceType)
		}
		if len(manifest.ModelIDs) == 0 {
			return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].modelIds 不能为空", idx)
		}
		seenModels := make(map[string]bool, len(manifest.ModelIDs))
		for modelIdx, modelID := range manifest.ModelIDs {
			if strings.TrimSpace(modelID) == "" {
				return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].modelIds[%d] 不能为空", idx, modelIdx)
			}
			if seenModels[modelID] {
				return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].modelIds[%d] 重复", idx, modelIdx)
			}
			seenModels[modelID] = true
		}
		for patternIdx, pattern := range manifest.ExcludeModelPatterns {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].excludeModelPatterns[%d] 不能为空", idx, patternIdx)
			}
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("[presetstore] builtinModelsManifests.manifests[%d].excludeModelPatterns[%d] 非法: %w", idx, patternIdx, err)
			}
		}
	}
	return nil
}

func validateNonNegativeInt(field string, value int) error {
	if value < 0 {
		return fmt.Errorf("%s 不能为负数", field)
	}
	return nil
}

func validatePricePointer(field string, value *float64) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s 不能为负数", field)
	}
	return nil
}

package presetstore

import "sync/atomic"

// 进程级默认 PresetStore。
//
// 包级消费点（如 autopilot.InferOriginTier、config.LookupBuiltinManifest）
// 通过 Default() 读取，避免层层透传 store 引用。main.go 在启动时用
// SetDefault 注入正式实例（含磁盘缓存/远程更新）；未注入时惰性回退到编译期内置，
// 保证测试与工具类命令无需初始化也能安全读取。
var defaultStore atomic.Pointer[PresetStore]

// Default 返回进程级默认 store；首次访问且未 SetDefault 时惰性初始化为内置默认。
func Default() *PresetStore {
	if s := defaultStore.Load(); s != nil {
		return s
	}
	// 惰性初始化：多 goroutine 竞争时用 CompareAndSwap 保证只装一次。
	s := NewPresetStore(EmbeddedBundle())
	if defaultStore.CompareAndSwap(nil, s) {
		return s
	}
	return defaultStore.Load()
}

// SetDefault 注入进程级默认 store（由 main.go 在启动时调用）。
func SetDefault(s *PresetStore) {
	if s != nil {
		defaultStore.Store(s)
	}
}

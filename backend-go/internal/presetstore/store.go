package presetstore

import (
	"sync"
	"sync/atomic"
)

// PresetStore 是运行时预置数据的并发安全存储。
//
// 读路径（Get）无锁，走 atomic.Pointer 快照；写路径（Swap）原子替换整个 bundle，
// 保证 Scheduler/Profiler 读到的始终是完整一致的一份数据，不会出现半更新态。
// 变更时同步触发已注册的观察者回调（复用 config.RegisterOnConfigChange 同款模式）。
type PresetStore struct {
	current atomic.Pointer[PresetBundle]

	mu        sync.Mutex
	observers []func(*PresetBundle)
}

// NewPresetStore 用给定初始 bundle 构造存储；initial 为 nil 时回退到编译期内置。
func NewPresetStore(initial *PresetBundle) *PresetStore {
	if initial == nil {
		initial = EmbeddedBundle()
	}
	s := &PresetStore{}
	s.current.Store(initial)
	return s
}

// Get 返回当前生效的 bundle 快照（只读，调用方不得原地修改）。
func (s *PresetStore) Get() *PresetBundle {
	return s.current.Load()
}

// Subscription 是 Get().Subscription 的便捷读取。
func (s *PresetStore) Subscription() SubscriptionPreset {
	return s.current.Load().Subscription
}

// DataVersion 返回当前生效数据版本（内置默认为空串）。
func (s *PresetStore) DataVersion() string {
	return s.current.Load().DataVersion
}

// Swap 原子替换当前 bundle 并触发观察者回调。
// 调用方须确保 next 已通过校验（见 Validate）。next 为 nil 时忽略。
func (s *PresetStore) Swap(next *PresetBundle) {
	if next == nil {
		return
	}
	s.current.Store(next)

	s.mu.Lock()
	observers := make([]func(*PresetBundle), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, cb := range observers {
		cb(next)
	}
}

// RegisterOnChange 注册数据变更回调（在 Swap 成功后触发）。
func (s *PresetStore) RegisterOnChange(cb func(*PresetBundle)) {
	if cb == nil {
		return
	}
	s.mu.Lock()
	s.observers = append(s.observers, cb)
	s.mu.Unlock()
}

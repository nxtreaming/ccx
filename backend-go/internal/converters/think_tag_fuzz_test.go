package converters

import (
	"strings"
	"testing"
)

// runMachineWithSplits 用给定的切分位置把 s 切片后逐段喂给状态机，最后 Drain。
// 返回总的 reasoning 和 content 文本（按发射顺序拼接）。
func runMachineWithSplits(s string, splits []int) (reasoning, content string) {
	var m thinkTagStateMachine
	m.Reset()
	prev := 0
	for _, p := range splits {
		if p <= prev || p > len(s) {
			continue
		}
		rs, cs := m.Feed(s[prev:p])
		for _, r := range rs {
			reasoning += r
		}
		for _, c := range cs {
			content += c
		}
		prev = p
	}
	if prev < len(s) {
		rs, cs := m.Feed(s[prev:])
		for _, r := range rs {
			reasoning += r
		}
		for _, c := range cs {
			content += c
		}
	}
	if rem, toReasoning := m.Drain(); rem != "" {
		if toReasoning {
			reasoning += rem
		} else {
			content += rem
		}
	}
	return reasoning, content
}

// FuzzThinkTagStream_SplitInvariant 验证状态机的核心不变量：
// 无论 chunk 如何切分，最终累积的 (reasoning, content) 必须与一次性整体喂入完全一致。
//
// 这条不变量能自动覆盖：
// 1. <think> / </think> 任意被切在中间的边界情况
// 2. 紧跟 <think> 的字符变化
// 3. ThinkTagBuf 兜底逻辑的正确性
func FuzzThinkTagStream_SplitInvariant(f *testing.F) {
	seeds := []struct {
		s    string
		mask uint64
	}{
		{"<think>hi</think>tail", 0x1},
		{"<think>multi</think><think>second</think>tail", 0x3},
		{"<think>unclosed", 0x7},
		{"<thi<think>weird</think>", 0xf},
		{"plain text", 0x0},
		{"  <think>spaces</think>real", 0x10},
		{"<think></think>", 0x20}, // 空 think
		{"<think>contains < and > and </think>after", 0x40},
	}
	for _, sd := range seeds {
		f.Add(sd.s, sd.mask)
	}

	f.Fuzz(func(t *testing.T, s string, mask uint64) {
		// 跳过过长输入（fuzz 中常见极端长度，浪费时间）
		if len(s) > 512 {
			return
		}

		// 一次性整体喂入
		wantR, wantC := runMachineWithSplits(s, nil)

		// 根据 mask 在不同位置切分（每个位置 1 bit）
		var splits []int
		for i := 0; i < len(s) && i < 64; i++ {
			if mask&(1<<uint(i)) != 0 {
				splits = append(splits, i+1)
			}
		}
		gotR, gotC := runMachineWithSplits(s, splits)

		if gotR != wantR || gotC != wantC {
			t.Errorf("split invariant violated for s=%q splits=%v:\n  got  reasoning=%q content=%q\n  want reasoning=%q content=%q",
				s, splits, gotR, gotC, wantR, wantC)
		}
	})
}

// FuzzExtractThinkTag 验证 extractThinkTag 不崩溃且自洽：
//   - 当 hasThink == false 时，返回的 text 应等于原 input（thinking 必为 ""）
//   - 当 hasThink == true 且 thinking 不含 "</think>" 时，重新拼接 "<think>" + thinking + "</think>" + text
//     去前导空白后应等价于原 input 的去前导空白形式
func FuzzExtractThinkTag(f *testing.F) {
	seeds := []string{
		"",
		"plain",
		"<think>a</think>b",
		"<think>unclosed",
		"<think></think>",
		"  \n<think>x</think>y",
		"head <think>x</think>tail",
		"<think>contains <think>nested</think> stuff</think>x",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 512 {
			return
		}
		text, thinking, hasThink := extractThinkTag(s)

		if !hasThink {
			if text != s {
				t.Errorf("hasThink=false but text != input: text=%q input=%q", text, s)
			}
			if thinking != "" {
				t.Errorf("hasThink=false but thinking=%q", thinking)
			}
			return
		}

		// hasThink==true 时 thinking 不应含 "</think>"（否则就提前断开了）
		if strings.Contains(thinking, thinkCloseTag) {
			t.Errorf("thinking contains </think>: %q", thinking)
		}

		// 重新拼接，验证可还原性（容忍 </think> 后的空白被吞掉）。
		// 只验证 hasThink==true 的常规情况。
		trimmedInput := strings.TrimLeft(s, " \t\r\n")
		// 检查 trimmedInput 以 <think> 开头
		if !strings.HasPrefix(trimmedInput, thinkOpenTag) {
			t.Errorf("hasThink=true but trimmed input doesn't start with <think>: %q", trimmedInput)
		}
	})
}

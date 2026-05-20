package converters

import "strings"

// think_tag.go 提供从模型输出（流式 / 非流式）中识别 <think>...</think>
// 推理块的字符级状态机与一次性提取函数。
// 这里只关心字符串切分，不涉及 SSE 事件发射，便于单测与复用。

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

// thinkTagState 状态机阶段。
type thinkTagState int

const (
	thinkStateNone   thinkTagState = iota // 等待开头的 <think>
	thinkStateInside                      // 在 <think>...</think> 内
	thinkStateDone                        // 已经匹配过一次，剩余都是正文
)

// thinkTagStateMachine 字符级状态机，识别跨 chunk 边界的 <think>...</think>。
//
// 仅允许在响应起始位置触发 <think>（CanStart）：一旦在非起始位置出现 "<think>" 或
// 其前缀，立即关闭检测窗口，剩余流量按普通文本透传，避免误判正文中的 "<think>"。
//
// 流结束时若 ThinkTagBuf 仍有内容（如 "<thi" 或 "</thi"），由调用方通过 Drain 取走兜底。
type thinkTagStateMachine struct {
	State     thinkTagState
	Buf       strings.Builder // 缓存跨 chunk 边界的不完整标签片段
	CanStart  bool            // 仅在响应起始位置允许触发 <think>
	LeadingWS strings.Builder // 缓存开头的空白字符，如果最终匹配到 <think> 则丢弃，否则作为正文输出
}

// Reset 把状态机恢复到初始状态。FirstChunk 路径应调用。
func (m *thinkTagStateMachine) Reset() {
	m.State = thinkStateNone
	m.Buf.Reset()
	m.CanStart = true
	m.LeadingWS.Reset()
}

// Feed 接收新 chunk，返回应分别送往 reasoning / content 通道的字符串切片。
// 状态机保留尾部可能的标签前缀（如 "<thi"）以便和下个 chunk 续接。
func (m *thinkTagStateMachine) Feed(chunk string) (reasoningParts, contentParts []string) {
	if chunk == "" {
		return nil, nil
	}
	pending := m.Buf.String() + chunk
	m.Buf.Reset()

	for len(pending) > 0 {
		switch m.State {
		case thinkStateNone: // 等待 <think>
			if !m.CanStart {
				contentParts = append(contentParts, pending)
				pending = ""
				continue
			}

			// 1. 提取前导空白字符
			var ws strings.Builder
			i := 0
			for i < len(pending) {
				c := pending[i]
				if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
					ws.WriteByte(c)
					i++
				} else {
					break
				}
			}
			if ws.Len() > 0 {
				m.LeadingWS.WriteString(ws.String())
				pending = pending[i:]
				if len(pending) == 0 {
					// 当前 chunk 全是空白，继续等待
					return nil, nil
				}
			}

			// 2. 检查 <think> 标签
			idx := strings.Index(pending, thinkOpenTag)
			if idx >= 0 {
				// idx > 0 说明 <think> 不在最开头（在空白字符之后还有其他非空白字符，然后才是 <think>）
				// 此时关闭检测窗口，把之前缓存的空白、当前 pending 里的内容全部作为正文
				if idx > 0 {
					if m.LeadingWS.Len() > 0 {
						contentParts = append(contentParts, m.LeadingWS.String())
						m.LeadingWS.Reset()
					}
					contentParts = append(contentParts, pending)
					pending = ""
					m.CanStart = false
					continue
				}
				// 匹配成功！丢弃前导空白，进入 thinkStateInside
				m.LeadingWS.Reset()
				m.State = thinkStateInside
				pending = pending[len(thinkOpenTag):]
				continue
			}

			// 未找到完整 <think>：若 pending 整体是 "<think>" 的非空前缀，保留到下个 chunk
			if isStrictPrefix(thinkOpenTag, pending) {
				m.Buf.WriteString(pending)
				return reasoningParts, contentParts
			}

			// 否则关闭检测窗口，把缓存的前导空白和当前 pending 全部作为正文
			if m.LeadingWS.Len() > 0 {
				contentParts = append(contentParts, m.LeadingWS.String())
				m.LeadingWS.Reset()
			}
			contentParts = append(contentParts, pending)
			pending = ""
			m.CanStart = false
		case thinkStateInside: // 等待 </think>
			idx := strings.Index(pending, thinkCloseTag)
			if idx >= 0 {
				if idx > 0 {
					reasoningParts = append(reasoningParts, pending[:idx])
				}
				pending = pending[idx+len(thinkCloseTag):]
				m.State = thinkStateDone
				continue
			}
			// 未找到完整 </think>：把末尾可能是 "</think>" 前缀的部分缓存
			keep := suffixThatCouldBePrefix(pending, thinkCloseTag)
			if keep > 0 {
				if len(pending) > keep {
					reasoningParts = append(reasoningParts, pending[:len(pending)-keep])
				}
				m.Buf.WriteString(pending[len(pending)-keep:])
				return reasoningParts, contentParts
			}
			reasoningParts = append(reasoningParts, pending)
			pending = ""
		case thinkStateDone: // 之后都是正文
			contentParts = append(contentParts, pending)
			pending = ""
		}
	}
	return reasoningParts, contentParts
}

// Drain 在流结束时取走状态机尾部缓冲。
// 返回 (剩余文本, 是否应进入 reasoning 通道)。无残留时返回 ("", false)。
func (m *thinkTagStateMachine) Drain() (remaining string, toReasoning bool) {
	if m.State == thinkStateNone {
		// 如果还在等待状态，需要把缓存的前导空白和 Buf 里的内容合并返回
		var sb strings.Builder
		if m.LeadingWS.Len() > 0 {
			sb.WriteString(m.LeadingWS.String())
			m.LeadingWS.Reset()
		}
		if m.Buf.Len() > 0 {
			sb.WriteString(m.Buf.String())
			m.Buf.Reset()
		}
		return sb.String(), false
	}

	if m.Buf.Len() == 0 {
		return "", false
	}
	remaining = m.Buf.String()
	m.Buf.Reset()
	// Inside：未闭合的 <think>... → reasoning；其他状态 → content
	toReasoning = m.State == thinkStateInside
	return remaining, toReasoning
}

// isStrictPrefix 报告 s 是否为 full 的非空严格前缀（s != full 且 full[:len(s)] == s）。
func isStrictPrefix(full, s string) bool {
	if s == "" || len(s) >= len(full) {
		return false
	}
	return full[:len(s)] == s
}

// suffixThatCouldBePrefix 返回 s 末尾最长的、可能成为 tag 严格前缀的长度。
// 例如 suffixThatCouldBePrefix("abc</thi", "</think>") == 5（"</thi"）。
func suffixThatCouldBePrefix(s, tag string) int {
	maxLen := len(s)
	if maxLen >= len(tag) {
		maxLen = len(tag) - 1
	}
	for k := maxLen; k > 0; k-- {
		if s[len(s)-k:] == tag[:k] {
			return k
		}
	}
	return 0
}

// extractThinkTag 从完整文本中提取开头位置的 <think>...</think>。
// 返回 (剩余文本, 思考内容, 是否检测到 think)。
// 仅在文本开头匹配，避免误判正文中的 "<think>"。未闭合的 <think> 视为全部为思考内容。
func extractThinkTag(content string) (text string, thinking string, hasThink bool) {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmed, thinkOpenTag) {
		return content, "", false
	}
	inner := trimmed[len(thinkOpenTag):]
	closeIdx := strings.Index(inner, thinkCloseTag)
	if closeIdx < 0 {
		return "", inner, true
	}
	thinking = inner[:closeIdx]
	remaining := inner[closeIdx+len(thinkCloseTag):]
	remaining = strings.TrimLeft(remaining, " \t\r\n")
	return remaining, thinking, true
}

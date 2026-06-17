export type ChannelServiceType = 'openai' | 'gemini' | 'claude' | 'responses' | ''
export type ReasoningEffort = 'none' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'
export type ReasoningParamStyle = 'reasoning' | 'reasoning_effort' | 'thinking'
export type TextVerbosity = 'low' | 'medium' | 'high' | ''

export interface AdvancedChannelOptions {
  reasoningMapping: Record<string, ReasoningEffort>
  reasoningParamStyle: ReasoningParamStyle
  textVerbosity: TextVerbosity
  fastMode: boolean
}

export const supportsAdvancedChannelOptions = (serviceType: ChannelServiceType): boolean => {
  return serviceType === 'openai' || serviceType === 'responses'
}

export const supportsReasoningMapping = (serviceType: ChannelServiceType): boolean => {
  return serviceType === 'openai' || serviceType === 'responses' || serviceType === 'claude'
}

export const normalizeAdvancedChannelOptions = (
  serviceType: ChannelServiceType,
  options: AdvancedChannelOptions
): AdvancedChannelOptions => {
  if (supportsAdvancedChannelOptions(serviceType)) {
    return options
  }

  return {
    reasoningMapping: supportsReasoningMapping(serviceType) ? options.reasoningMapping : {},
    reasoningParamStyle: 'reasoning',
    textVerbosity: '',
    fastMode: false
  }
}

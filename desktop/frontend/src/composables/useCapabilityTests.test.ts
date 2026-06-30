import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { AdminApiError } from '@/composables/useAdminApi'
import { useCapabilityTests } from './useCapabilityTests'
import type { CapabilityTestJob } from '@/services/admin-api'

const apiPost = vi.fn()
const apiGet = vi.fn()

vi.mock('@/composables/useAdminApi', async () => {
  const actual = await vi.importActual<typeof import('@/composables/useAdminApi')>('@/composables/useAdminApi')
  return {
    AdminApiError: actual.AdminApiError,
    useAdminApi: () => ({
      post: apiPost,
      get: apiGet,
      del: vi.fn(),
    }),
  }
})

vi.mock('@/composables/useConsoleChannels', () => ({
  useConsoleChannels: () => ({
    refreshChannels: vi.fn(),
  }),
}))

describe('useCapabilityTests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useCapabilityTests().reset()
  })

  it('falls back to a single-model protocol test when retry job does not contain that model', async () => {
    const capability = useCapabilityTests()
    capability.prepareChannelSession('messages', 1, 'channel')

    const completedJob = buildJob('job-1', [
      {
        model: 'claude-a',
        status: 'failed',
        lifecycle: 'done',
        outcome: 'failed',
        success: false,
        latency: 0,
        streamingSupported: false,
      },
      {
        model: 'claude-b',
        status: 'idle',
        lifecycle: 'pending',
        outcome: 'unknown',
        success: false,
        latency: 0,
        streamingSupported: false,
      },
    ])
    capability.activeJob.value = completedJob

    apiPost.mockRejectedValueOnce(new AdminApiError('Model not found in job', 404))
    apiPost.mockResolvedValueOnce({
      jobId: 'job-2',
      job: buildJob('job-2', [
        {
          model: 'claude-b',
          status: 'queued',
          lifecycle: 'pending',
          outcome: 'unknown',
          success: false,
          latency: 0,
          streamingSupported: false,
        },
      ], 'queued', 'pending'),
    })

    await capability.retryModelForProtocol('messages', 1, 'messages', 'claude-b')
    await nextTick()

    expect(apiPost).toHaveBeenNthCalledWith(
      1,
      '/api/messages/channels/1/capability-test/job-1/retry',
      { protocol: 'messages', model: 'claude-b' },
    )
    expect(apiPost).toHaveBeenNthCalledWith(
      2,
      '/api/messages/channels/1/capability-test',
      expect.objectContaining({
        targetProtocols: ['messages'],
        models: ['claude-b'],
        previousJobId: 'job-1',
      }),
    )
    const retriedModel = capability.activeJob.value?.tests
      .find(test => test.protocol === 'messages')
      ?.modelResults?.find(modelResult => modelResult.model === 'claude-b')
    expect(retriedModel?.status).toBe('queued')
  })
})

function buildJob(
  jobId: string,
  modelResults: CapabilityTestJob['tests'][number]['modelResults'],
  status: CapabilityTestJob['status'] = 'completed',
  lifecycle: CapabilityTestJob['lifecycle'] = 'done',
): CapabilityTestJob {
  return {
    jobId,
    channelId: 1,
    channelName: 'channel',
    channelKind: 'messages',
    sourceType: 'messages',
    status,
    lifecycle,
    outcome: status === 'completed' ? 'failed' : 'unknown',
    runMode: 'fresh',
    tests: [{
      protocol: 'messages',
      status: status === 'queued' ? 'queued' : 'failed',
      lifecycle,
      outcome: status === 'completed' ? 'failed' : 'unknown',
      success: false,
      latency: 0,
      streamingSupported: false,
      testedModel: '',
      modelResults,
      successCount: 0,
      attemptedModels: modelResults?.length ?? 0,
      testedAt: new Date().toISOString(),
    }],
    compatibleProtocols: [],
    totalDuration: 0,
    updatedAt: new Date().toISOString(),
    targetProtocols: ['messages'],
    protocolJobIds: { messages: jobId },
    protocolJobRefs: { messages: { jobId, channelKind: 'messages', channelId: 1 } },
    progress: {
      totalModels: modelResults?.length ?? 0,
      queuedModels: status === 'queued' ? modelResults?.length ?? 0 : 0,
      runningModels: 0,
      successModels: 0,
      failedModels: status === 'completed' ? modelResults?.length ?? 0 : 0,
      skippedModels: 0,
      completedModels: status === 'completed' ? modelResults?.length ?? 0 : 0,
    },
  }
}

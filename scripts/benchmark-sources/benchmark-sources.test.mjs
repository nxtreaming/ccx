import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canonicalModelToPattern,
  deepsweModelToPattern,
} from './mapper.mjs'
import {
  extractBestPerModel as extractDeepSWEBest,
  toBenchmarkEvidence as toDeepSWEEvidence,
} from './deepswe.mjs'
import {
  extractBestPerModel as extractDradarBest,
  toBenchmarkEvidence as toDradarEvidence,
} from './dradar.mjs'
import { extractModelInfo } from './litellm.mjs'
import {
  mergeDeepsweData,
  mergeLitellmData,
  validateRegistry,
} from '../update-benchmark-data.mjs'

function emptyReport() {
  return { updated: [], added: [], errors: [], litellmUpdated: [], litellmSkipped: [] }
}

test('canonical pattern generation accepts canonical and source model names', () => {
  const expected = '(?:^|[-/])gpt-5\\.6-sol(?=$|@)'
  assert.equal(canonicalModelToPattern('gpt-5.6-sol'), expected)
  assert.equal(deepsweModelToPattern('gpt-5-6-sol'), expected)
  assert.equal(deepsweModelToPattern('gpt-5.6-sol'), null)
})

test('DeepSWE percentile and cohort use one best row per model', () => {
  const rows = [
    { model: 'model-a', pass_at_1: 0.8, reasoning_effort: 'high', n_tasks_attempted: 100 },
    { model: 'model-a', pass_at_1: 0.7, reasoning_effort: 'low', n_tasks_attempted: 100 },
    { model: 'model-b', pass_at_1: 0.6, reasoning_effort: 'high', n_tasks_attempted: 100 },
  ]
  const best = extractDeepSWEBest({ rows }, { 'model-a': 'a', 'model-b': 'b' })
  const evidence = toDeepSWEEvidence(best[0], best)

  assert.equal(best.length, 2)
  assert.equal(evidence.cohortSize, 2)
  assert.equal(evidence.cohortPercentile, 1)
  assert.equal(evidence.taskCount, 100)
})

test('dradar cohort size is model count rather than graded run count', () => {
  const best = extractDradarBest({
    models: [
      { model: 'a', effort: 'high', pass_rate: 0.8, graded: 450, cells: 100, cells_passed: 80 },
      { model: 'b', effort: 'high', pass_rate: 0.6, graded: 440, cells: 100, cells_passed: 60 },
    ],
  }, { a: 'a', b: 'b' })
  const evidence = toDradarEvidence(best.a, Object.values(best))

  assert.equal(evidence.cohortSize, 2)
  assert.equal(evidence.cohortPercentile, 1)
})

test('LiteLLM keeps missing capabilities unknown and maps function calling to toolCalls', () => {
  const info = extractModelInfo({
    source: {
      max_input_tokens: 100_000,
      supports_function_calling: true,
    },
  }, { source: 'canonical' }).canonical

  assert.equal(info.supports.toolCalls, true)
  assert.equal(info.supports.vision, undefined)
  assert.equal(info.supports.reasoning, undefined)
  assert.equal(Object.hasOwn(info.supports, 'functionCalling'), false)
})

test('benchmark merge creates a complete valid profile', () => {
  const registry = { benchmarkProfiles: [], upstreamCapabilities: [] }
  mergeDeepsweData(registry, {
    'gpt-5.6-sol': {
      deepsweMeta: { deepsweModel: 'gpt-5-6-sol' },
      benchmarkEvidence: [{
        benchmark: 'deepswe',
        benchmarkVersion: 'v1.1',
        sourceModel: 'gpt-5-6-sol',
        domain: 'coding',
        metric: 'pass_at_1',
        rawValue: 0.8,
        uncertainty: 0.01,
        cohortPercentile: 1,
        taskCount: 100,
        cohortSize: 4,
        effort: 'high',
        selectionBasis: 'best_available_effort',
        sourceUrl: 'https://deepswe.example/',
        capturedAt: '2026-07-21',
      }],
    },
  }, emptyReport(), null)

  assert.doesNotThrow(() => validateRegistry(registry))
  assert.deepEqual(registry.benchmarkProfiles[0].sources, ['https://deepswe.example/'])
  assert.equal(registry.benchmarkProfiles[0].sharedResults, 4)
})

test('LiteLLM fills only unknown capabilities', () => {
  const registry = {
    upstreamCapabilities: [{
      patterns: ['(?:^|[-/])model(?=$|@)'],
      capabilities: { vision: true },
    }],
  }
  mergeLitellmData(registry, {
    model: { supports: { vision: false, toolCalls: true } },
  }, emptyReport(), null)

  assert.equal(registry.upstreamCapabilities[0].capabilities.vision, true)
  assert.equal(registry.upstreamCapabilities[0].capabilities.toolCalls, true)
})

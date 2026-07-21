/**
 * 模型能力基准自动更新编排脚本
 *
 * 功能：
 * 1. 从 deepswe、benchlm.ai、dradar (codexradar) 抓取最新 benchmark 数据
 * 2. 从 litellm 抓取价格/上下文窗口数据
 * 3. 映射到 CCX 模型注册表
 * 4. 更新 shared/model-registry/ccx_model_registry.json
 * 5. 运行 generate-model-registry.mjs 重新生成代码
 * 6. 输出变更报告
 *
 * 用法：
 *   node scripts/update-benchmark-data.mjs [--dry-run] [--skip-*] [--models <model1,model2>]
 *
 * 选项：
 *   --dry-run       只预览变更，不写入文件
 *   --skip-deepswe  跳过 deepswe 数据源
 *   --skip-benchlm  跳过 benchlm.ai 数据源
 *   --skip-dradar   跳过 dradar (codexradar) 数据源
 *   --skip-litellm  跳过 litellm 价格/上下文数据源
 *   --models        只更新指定模型 (逗号分隔)
 */

import { existsSync, readFileSync, renameSync, unlinkSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

import {
  DEEPSWE_MODEL_MAP,
  BENCHLM_MODEL_MAP,
  BENCHLM_CATEGORY_MAP,
  canonicalModelToPattern,
  deepsweModelToPattern,
} from './benchmark-sources/mapper.mjs'
import { fetchDeepsweData } from './benchmark-sources/deepswe.mjs'
import { fetchBenchlmData } from './benchmark-sources/benchlm.mjs'
import { fetchDradarData, DRADAR_MODEL_MAP } from './benchmark-sources/dradar.mjs'
import { fetchLitellmModelInfo, LITELLM_MODEL_MAP } from './benchmark-sources/litellm.mjs'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const registryPath = join(root, 'shared/model-registry/ccx_model_registry.json')

// 命令行参数
const args = process.argv.slice(2)
const dryRun = args.includes('--dry-run')
const skipDeepswe = args.includes('--skip-deepswe')
const skipBenchlm = args.includes('--skip-benchlm')
const skipDradar = args.includes('--skip-dradar')
const skipLitellm = args.includes('--skip-litellm')
const modelsArg = args.find(a => a.startsWith('--models='))
const modelsArgIndex = args.indexOf('--models')
const modelsValue = modelsArg?.split('=', 2)[1] ?? (modelsArgIndex >= 0 ? args[modelsArgIndex + 1] : '')
const targetModels = modelsValue ? modelsValue.split(',').map(model => model.trim()).filter(Boolean) : null
const generatedPaths = [
  join(root, 'backend-go/internal/config/generated_model_registry.go'),
  join(root, 'frontend/src/generated/modelRegistry.ts'),
  join(root, 'desktop/frontend/src/generated/model-registry.ts'),
]

/**
 * 加载注册表
 */
function loadRegistry() {
  const content = readFileSync(registryPath, 'utf8')
  return JSON.parse(content)
}

/**
 * 保存注册表
 */
function serializeRegistry(registry) {
  return JSON.stringify(registry, null, 2) + '\n'
}

function atomicWrite(path, content) {
  const tempPath = `${path}.tmp-${process.pid}-${Date.now()}`
  try {
    writeFileSync(tempPath, content, 'utf8')
    renameSync(tempPath, path)
  } catch (error) {
    if (existsSync(tempPath)) unlinkSync(tempPath)
    throw error
  }
}

function saveRegistry(registry) {
  atomicWrite(registryPath, serializeRegistry(registry))
}

/**
 * 查找现有的 benchmarkProfile
 * @param {Array} profiles
 * @param {string} canonicalModel
 * @returns {number} - 索引，未找到返回 -1
 */
function findProfileIndex(profiles, canonicalModel) {
  return profiles.findIndex(p => p.canonicalModel === canonicalModel)
}

/**
 * 创建新的 benchmarkProfile
 * @param {string} canonicalModel
 * @param {string} pattern
 * @returns {Object}
 */
export function createProfile(canonicalModel, pattern = canonicalModelToPattern(canonicalModel)) {
  if (!pattern) {
    throw new Error(`cannot generate benchmark pattern for ${canonicalModel}`)
  }
  return {
    patterns: [pattern],
    canonicalModel,
    verifiedAt: new Date().toISOString().split('T')[0],
    lane: 'provisional',
    sources: [],
    sharedResults: 1,
    comparableCategories: 1,
    totalCategories: 1,
  }
}

function ensureEvidenceProfileMetadata(profile) {
  const evidence = profile.benchmarkEvidence || []
  const sourceURLs = evidence.map(item => item.sourceUrl).filter(Boolean)
  profile.sources = [...new Set([...(profile.sources || []), ...sourceURLs])]

  const cohortSize = Math.max(0, ...evidence.map(item => Number(item.cohortSize) || 0))
  const domainCount = new Set(evidence.map(item => item.domain).filter(Boolean)).size
  profile.sharedResults = Math.max(Number(profile.sharedResults) || 0, cohortSize, 1)
  profile.comparableCategories = Math.max(Number(profile.comparableCategories) || 0, domainCount, 1)
  profile.totalCategories = Math.max(
    Number(profile.totalCategories) || 0,
    profile.comparableCategories,
  )
}

/**
 * 合并 deepswe 数据到注册表
 */
export function mergeDeepsweData(registry, deepsweData, report, models = targetModels) {
  if (!registry.benchmarkProfiles) {
    registry.benchmarkProfiles = []
  }

  for (const [canonical, data] of Object.entries(deepsweData)) {
    if (models && !models.includes(canonical)) {
      continue
    }

    const idx = findProfileIndex(registry.benchmarkProfiles, canonical)
    const profile = idx >= 0
      ? registry.benchmarkProfiles[idx]
      : createProfile(
          canonical,
          deepsweModelToPattern(data.deepsweMeta?.deepsweModel) || canonicalModelToPattern(canonical),
        )

    // 确保 benchmarkEvidence 存在
    if (!profile.benchmarkEvidence) {
      profile.benchmarkEvidence = []
    }

    // 移除旧的 deepswe 证据
    profile.benchmarkEvidence = profile.benchmarkEvidence.filter(e => e.benchmark !== 'deepswe')

    // 添加新的 deepswe 证据
    profile.benchmarkEvidence.push(...data.benchmarkEvidence)
    ensureEvidenceProfileMetadata(profile)

    // 更新 verifiedAt
    profile.verifiedAt = new Date().toISOString().split('T')[0]

    // 更新或插入
    if (idx >= 0) {
      registry.benchmarkProfiles[idx] = profile
      report.updated.push({ canonical, source: 'deepswe' })
    } else {
      registry.benchmarkProfiles.push(profile)
      report.added.push({ canonical, source: 'deepswe' })
    }
  }
}

/**
 * 合并 benchlm.ai 数据到注册表
 */
export function mergeBenchlmData(registry, benchlmData, report, models = targetModels) {
  if (!registry.benchmarkProfiles) {
    registry.benchmarkProfiles = []
  }

  for (const [canonical, data] of Object.entries(benchlmData)) {
    if (models && !models.includes(canonical)) {
      continue
    }

    const idx = findProfileIndex(registry.benchmarkProfiles, canonical)
    const profile = idx >= 0 ? registry.benchmarkProfiles[idx] : createProfile(canonical)

    // 更新 overallScore
    if (data.overallScore !== null && data.overallScore !== undefined) {
      profile.overallScore = data.overallScore
    }

    // 更新 categoryScores
    if (Object.keys(data.categoryScores).length > 0) {
      profile.categoryScores = data.categoryScores
    }

    // 更新 counts
    if (data.counts) {
      profile.sharedResults = data.counts.sharedBenchmarkCount
      profile.comparableCategories = data.counts.comparableCategoryCount
      profile.totalCategories = data.counts.totalCategoryCount
    }

    // 更新 sources
    if (data.sources && data.sources.length > 0) {
      const existingSources = profile.sources || []
      const newSources = [...new Set([...existingSources, ...data.sources])]
      profile.sources = newSources
    }

    // 更新 verifiedAt
    profile.verifiedAt = new Date().toISOString().split('T')[0]

    // 更新或插入
    if (idx >= 0) {
      registry.benchmarkProfiles[idx] = profile
      report.updated.push({ canonical, source: 'benchlm' })
    } else {
      registry.benchmarkProfiles.push(profile)
      report.added.push({ canonical, source: 'benchlm' })
    }
  }
}

/**
 * 合并 dradar 数据到注册表
 */
export function mergeDradarData(registry, dradarData, report, models = targetModels) {
  if (!registry.benchmarkProfiles) {
    registry.benchmarkProfiles = []
  }

  for (const [canonical, data] of Object.entries(dradarData)) {
    if (models && !models.includes(canonical)) {
      continue
    }

    const idx = findProfileIndex(registry.benchmarkProfiles, canonical)
    const profile = idx >= 0 ? registry.benchmarkProfiles[idx] : createProfile(canonical)

    // 确保 benchmarkEvidence 存在
    if (!profile.benchmarkEvidence) {
      profile.benchmarkEvidence = []
    }

    // 移除旧的 deepswe-codexradar 证据（dradar 的 benchmark 是 deepswe，但版本是 codexradar）
    profile.benchmarkEvidence = profile.benchmarkEvidence.filter(
      e => !(e.benchmark === 'deepswe' && e.benchmarkVersion === 'codexradar')
    )

    // 添加新的 dradar 证据
    profile.benchmarkEvidence.push(...data.benchmarkEvidence)
    ensureEvidenceProfileMetadata(profile)

    // 添加 cost 数据作为额外字段（如果 profile 支持）
    if (data.costData && Object.keys(data.costData).length > 0) {
      if (!profile.costData) {
        profile.costData = {}
      }
      profile.costData = data.costData
    }

    // 更新 verifiedAt
    profile.verifiedAt = new Date().toISOString().split('T')[0]

    // 更新或插入
    if (idx >= 0) {
      registry.benchmarkProfiles[idx] = profile
      report.updated.push({ canonical, source: 'dradar' })
    } else {
      registry.benchmarkProfiles.push(profile)
      report.added.push({ canonical, source: 'dradar' })
    }
  }
}

/**
 * 合并 litellm 数据到注册表（更新 upstreamCapabilities 的 pricing/contextWindow）
 */
export function mergeLitellmData(registry, litellmData, report, models = targetModels) {
  if (!registry.upstreamCapabilities) {
    registry.upstreamCapabilities = []
  }

  for (const [canonical, data] of Object.entries(litellmData)) {
    if (models && !models.includes(canonical)) {
      continue
    }

    // 查找现有的 upstreamCapability
    const capIdx = registry.upstreamCapabilities.findIndex(c => {
      const patterns = c.patterns || []
      return patterns.some(p => {
        // 检查 pattern 是否匹配 canonical 模型名
        try {
          const regex = new RegExp(p, 'i')
          return regex.test(canonical)
        } catch {
          return false
        }
      })
    })

    if (capIdx >= 0) {
      const cap = registry.upstreamCapabilities[capIdx]

      // 更新 contextWindowTokens
      if (data.contextWindowTokens && !cap.contextWindowTokens) {
        cap.contextWindowTokens = data.contextWindowTokens
        report.litellmUpdated.push({ canonical, field: 'contextWindowTokens', value: data.contextWindowTokens })
      }

      // 更新 maxOutputTokens
      if (data.maxOutputTokens && !cap.maxOutputTokens) {
        cap.maxOutputTokens = data.maxOutputTokens
        report.litellmUpdated.push({ canonical, field: 'maxOutputTokens', value: data.maxOutputTokens })
      }

      // 更新 pricing（如果 litellm 有数据）
      if (data.pricing && Object.values(data.pricing).some(v => v !== null)) {
        if (!cap.pricing) {
          cap.pricing = { unit: 'per_1m_tokens_usd', currency: 'USD' }
        }
        if (data.pricing.inputCacheMissPrice !== null) {
          cap.pricing.inputCacheMissPrice = data.pricing.inputCacheMissPrice
        }
        if (data.pricing.outputPrice !== null) {
          cap.pricing.outputPrice = data.pricing.outputPrice
        }
        if (data.pricing.inputCacheHitPrice !== null) {
          cap.pricing.inputCacheHitPrice = data.pricing.inputCacheHitPrice
        }
        report.litellmUpdated.push({ canonical, field: 'pricing', value: 'updated' })
      }

      // 更新 capabilities
      if (data.supports) {
        if (!cap.capabilities) {
          cap.capabilities = {}
        }
        let capabilitiesUpdated = false
        for (const field of ['reasoning', 'vision', 'toolCalls', 'parallelFunctionCalling', 'webSearch']) {
          if (data.supports[field] !== undefined && cap.capabilities[field] === undefined) {
            cap.capabilities[field] = data.supports[field]
            capabilitiesUpdated = true
          }
        }
        if (capabilitiesUpdated) {
          report.litellmUpdated.push({ canonical, field: 'capabilities', value: 'filled missing values' })
        }
      }

      registry.upstreamCapabilities[capIdx] = cap
    } else {
      report.litellmSkipped.push({ canonical, reason: 'not found in upstreamCapabilities' })
    }
  }
}

export function validateRegistry(registry) {
  for (const [index, profile] of (registry.benchmarkProfiles || []).entries()) {
    const prefix = `benchmarkProfiles[${index}]`
    if (!profile.canonicalModel || !Array.isArray(profile.patterns) || profile.patterns.length === 0) {
      throw new Error(`${prefix} is missing canonicalModel or patterns`)
    }
    if (profile.patterns.some(pattern => typeof pattern !== 'string' || pattern.trim() === '')) {
      throw new Error(`${prefix} contains an empty pattern`)
    }
    if ((!profile.categoryScores || Object.keys(profile.categoryScores).length === 0) &&
        (!profile.benchmarkEvidence || profile.benchmarkEvidence.length === 0)) {
      throw new Error(`${prefix} requires categoryScores or benchmarkEvidence`)
    }
    if (!Array.isArray(profile.sources) || profile.sources.length === 0) {
      throw new Error(`${prefix} requires at least one source`)
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(profile.verifiedAt || '')) {
      throw new Error(`${prefix}.verifiedAt must use YYYY-MM-DD`)
    }
    if (!['provisional', 'verified'].includes(profile.lane)) {
      throw new Error(`${prefix}.lane is invalid`)
    }
    for (const field of ['sharedResults', 'comparableCategories', 'totalCategories']) {
      if (!Number.isFinite(profile[field]) || profile[field] <= 0) {
        throw new Error(`${prefix}.${field} must be positive`)
      }
    }
    if (profile.comparableCategories > profile.totalCategories) {
      throw new Error(`${prefix}.comparableCategories exceeds totalCategories`)
    }
  }
}

/**
 * 运行代码生成
 */
function runCodeGeneration() {
  console.log('\n[generate] Running generate-model-registry.mjs...')
  try {
    execFileSync('node', [join(root, 'scripts/generate-model-registry.mjs')], {
      stdio: 'inherit',
      cwd: root,
    })
    console.log('[generate] Done')
  } catch (err) {
    console.error('[generate] Failed:', err.message)
    throw err
  }
}

function saveAndGenerateAtomically(registry) {
  const trackedPaths = [registryPath, ...generatedPaths]
  const snapshots = new Map(
    trackedPaths
      .filter(path => existsSync(path))
      .map(path => [path, readFileSync(path, 'utf8')]),
  )

  try {
    saveRegistry(registry)
    runCodeGeneration()
  } catch (error) {
    for (const [path, content] of snapshots) {
      atomicWrite(path, content)
    }
    throw error
  }
}

/**
 * 主函数
 */
export async function main() {
  console.log('='.repeat(60))

  if (skipDeepswe && skipBenchlm && skipDradar && skipLitellm) {
    throw new Error('all benchmark sources are skipped')
  }
  console.log('CCX Benchmark Data Auto-Updater')
  console.log('='.repeat(60))
  console.log(`Mode: ${dryRun ? 'DRY RUN' : 'UPDATE'}`)
  console.log(`Registry: ${registryPath}`)
  console.log(`Skip deepswe: ${skipDeepswe}`)
  console.log(`Skip benchlm: ${skipBenchlm}`)
  console.log(`Skip dradar: ${skipDradar}`)
  console.log(`Skip litellm: ${skipLitellm}`)
  if (targetModels) {
    console.log(`Target models: ${targetModels.join(', ')}`)
  }
  console.log('='.repeat(60))

  const registry = loadRegistry()
  const report = {
    updated: [],
    added: [],
    errors: [],
    litellmUpdated: [],
    litellmSkipped: [],
  }

  // 抓取 deepswe 数据
  if (!skipDeepswe) {
    try {
      console.log('\n--- Fetching deepswe data ---')
      const deepsweData = await fetchDeepsweData(DEEPSWE_MODEL_MAP)
      mergeDeepsweData(registry, deepsweData, report)
    } catch (err) {
      report.errors.push({ source: 'deepswe', error: err.message })
      console.error('[deepswe] Failed:', err.message)
    }
  }

  // 抓取 benchlm.ai 数据
  if (!skipBenchlm) {
    try {
      console.log('\n--- Fetching benchlm.ai data ---')
      const benchlmData = await fetchBenchlmData(BENCHLM_MODEL_MAP, BENCHLM_CATEGORY_MAP)
      mergeBenchlmData(registry, benchlmData, report)
    } catch (err) {
      report.errors.push({ source: 'benchlm', error: err.message })
      console.error('[benchlm] Failed:', err.message)
    }
  }

  // 抓取 dradar (codexradar) 数据
  if (!skipDradar) {
    try {
      console.log('\n--- Fetching dradar (codexradar) data ---')
      const dradarData = await fetchDradarData(DRADAR_MODEL_MAP)
      mergeDradarData(registry, dradarData, report)
    } catch (err) {
      report.errors.push({ source: 'dradar', error: err.message })
      console.error('[dradar] Failed:', err.message)
    }
  }

  // 抓取 litellm 数据
  if (!skipLitellm) {
    try {
      console.log('\n--- Fetching litellm pricing/context data ---')
      const litellmData = await fetchLitellmModelInfo(LITELLM_MODEL_MAP)
      mergeLitellmData(registry, litellmData, report)
    } catch (err) {
      report.errors.push({ source: 'litellm', error: err.message })
      console.error('[litellm] Failed:', err.message)
    }
  }

  if (report.errors.length > 0) {
    const failedSources = report.errors.map(item => item.source).join(', ')
    throw new Error(`enabled sources failed (${failedSources}); registry was not changed`)
  }

  validateRegistry(registry)

  // 保存注册表
  if (!dryRun) {
    console.log('\n--- Saving registry ---')
    saveAndGenerateAtomically(registry)
    console.log(`[save] Registry and generated code updated atomically`)
  } else {
    console.log('\n--- DRY RUN: No changes saved ---')
  }

  // 输出报告
  console.log('\n' + '='.repeat(60))
  console.log('UPDATE REPORT')
  console.log('='.repeat(60))
  console.log(`Updated profiles: ${report.updated.length}`)
  for (const u of report.updated) {
    console.log(`  - ${u.canonical} (${u.source})`)
  }
  console.log(`Added profiles: ${report.added.length}`)
  for (const a of report.added) {
    console.log(`  + ${a.canonical} (${a.source})`)
  }
  if (report.errors.length > 0) {
    console.log(`Errors: ${report.errors.length}`)
    for (const e of report.errors) {
      console.log(`  ! ${e.source}: ${e.error}`)
    }
  }
  if (report.litellmUpdated.length > 0) {
    console.log(`\nlitellm updates: ${report.litellmUpdated.length}`)
    for (const u of report.litellmUpdated.slice(0, 10)) {
      console.log(`  - ${u.canonical}.${u.field}: ${u.value}`)
    }
    if (report.litellmUpdated.length > 10) {
      console.log(`  ... and ${report.litellmUpdated.length - 10} more`)
    }
  }
  if (report.litellmSkipped.length > 0) {
    console.log(`\nlitellm skipped: ${report.litellmSkipped.length}`)
    for (const s of report.litellmSkipped.slice(0, 5)) {
      console.log(`  - ${s.canonical}: ${s.reason}`)
    }
  }
  console.log('='.repeat(60))

  if (dryRun) {
    console.log('\nTo apply changes, run without --dry-run')
  }
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : ''
if (invokedPath === fileURLToPath(import.meta.url)) {
  main().catch(err => {
    console.error('Fatal error:', err)
    process.exit(1)
  })
}

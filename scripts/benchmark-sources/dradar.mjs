/**
 * dradar (codexradar) 数据抓取器
 *
 * 从 https://api.codexradar.com 抓取 leaderboard 数据。
 *
 * 数据来源：
 * - /api/v1/leaderboard: 各模型+effort 的 pass_rate、graded 数
 * - /api/v1/table: 各 task+model+effort 的 cell 数据（含 cost）
 *
 * 数据结构：
 * - leaderboard: {models: [{model, effort, pass_rate, graded, passed, cells, cells_passed, tasks}]}
 * - table: {cells: {"task|model|effort": {rate, n, p, ran_by: [{actual_cost_usd, duration_sec, ...}]}}}
 */

/**
 * dradar 模型名 -> CCX canonicalModel 映射
 * dradar 使用点号: "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5"
 */
import { fetchWithTimeout } from './http.mjs'

export const DRADAR_MODEL_MAP = {
  'gpt-5.6-sol': 'gpt-5.6-sol',
  'gpt-5.6-terra': 'gpt-5.6-terra',
  'gpt-5.6-luna': 'gpt-5.6-luna',
  'gpt-5.5': 'gpt-5.5',
}

const BASE_URL = 'https://api.codexradar.com'

/**
 * 获取 dradar leaderboard 数据
 * @returns {Promise<Object>}
 */
export async function fetchLeaderboard() {
  const url = `${BASE_URL}/api/v1/leaderboard`

  console.log(`[dradar] Fetching ${url}`)

  const resp = await fetchWithTimeout(url, {
    headers: {
      'User-Agent': 'ccx-benchmark-updater/1.0',
      Accept: 'application/json',
    },
  })

  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status} ${resp.statusText} for ${url}`)
  }

  return resp.json()
}

/**
 * 获取 dradar table 数据（含 cost）
 * @returns {Promise<Object>}
 */
export async function fetchTable() {
  const url = `${BASE_URL}/api/v1/table?ui=v1`

  console.log(`[dradar] Fetching ${url}`)

  const resp = await fetchWithTimeout(url, {
    headers: {
      'User-Agent': 'ccx-benchmark-updater/1.0',
      Accept: 'application/json',
    },
  })

  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status} ${resp.statusText} for ${url}`)
  }

  return resp.json()
}

/**
 * 从 leaderboard 数据中提取每个模型的最佳表现
 *
 * @param {Object} data - leaderboard JSON 数据
 * @param {Object} modelMap - dradar 模型名 -> CCX canonicalModel 映射
 * @returns {Object} - {canonicalModel: {bestEffort, passRate, graded, cells, cellsPassed, efforts}}
 */
export function extractBestPerModel(data, modelMap) {
  const models = data.models || []
  const result = {}

  for (const m of models) {
    const dradarModel = m.model
    const canonical = modelMap[dradarModel]

    if (!canonical) {
      continue
    }

    if (!result[canonical]) {
      result[canonical] = {
        canonicalModel: canonical,
        deepsweModel: dradarModel,
        bestEffort: null,
        passRate: 0,
        graded: 0,
        cells: 0,
        cellsPassed: 0,
        efforts: {},
      }
    }

    result[canonical].efforts[m.effort] = {
      passRate: m.pass_rate,
      graded: m.graded,
      cells: m.cells,
      cellsPassed: m.cells_passed,
    }

    // 更新最佳 effort
    if (m.pass_rate > result[canonical].passRate) {
      result[canonical].bestEffort = m.effort
      result[canonical].passRate = m.pass_rate
      result[canonical].graded = m.graded
      result[canonical].cells = m.cells
      result[canonical].cellsPassed = m.cells_passed
    }
  }

  return result
}

/**
 * 从 table 数据中提取 cost 信息
 *
 * @param {Object} data - table JSON 数据
 * @param {Object} modelMap - dradar 模型名 -> CCX canonicalModel 映射
 * @returns {Object} - {canonicalModel: {effort: {meanCost, medianCost, nRuns}}}
 */
export function extractCostData(data, modelMap) {
  const cells = data.cells || {}
  const costByModelEffort = {}

  for (const [key, cell] of Object.entries(cells)) {
    const [taskId, model, effort] = key.split('|')
    const canonical = modelMap[model]

    if (!canonical || !cell.ran_by || cell.ran_by.length === 0) {
      continue
    }

    if (!costByModelEffort[canonical]) {
      costByModelEffort[canonical] = {}
    }
    if (!costByModelEffort[canonical][effort]) {
      costByModelEffort[canonical][effort] = {
        costs: [],
        durations: [],
      }
    }

    for (const run of cell.ran_by) {
      if (run.actual_cost_usd !== null && run.actual_cost_usd !== undefined) {
        costByModelEffort[canonical][effort].costs.push(run.actual_cost_usd)
      }
      if (run.duration_sec !== null && run.duration_sec !== undefined) {
        costByModelEffort[canonical][effort].durations.push(run.duration_sec)
      }
    }
  }

  // 计算均值和中位数
  const result = {}
  for (const [canonical, efforts] of Object.entries(costByModelEffort)) {
    result[canonical] = {}
    for (const [effort, data] of Object.entries(efforts)) {
      if (data.costs.length === 0) continue

      const sortedCosts = [...data.costs].sort((a, b) => a - b)
      const sortedDurations = [...data.durations].sort((a, b) => a - b)

      result[canonical][effort] = {
        meanCost: data.costs.reduce((a, b) => a + b, 0) / data.costs.length,
        medianCost: sortedCosts[Math.floor(sortedCosts.length / 2)],
        meanDuration: data.durations.length > 0 ? data.durations.reduce((a, b) => a + b, 0) / data.durations.length : null,
        medianDuration: data.durations.length > 0 ? sortedDurations[Math.floor(sortedDurations.length / 2)] : null,
        nRuns: data.costs.length,
      }
    }
  }

  return result
}

/**
 * 生成 benchmarkEvidence 对象
 * @param {Object} modelData - extractBestPerModel 的输出
 * @param {Array} allModels - 所有模型列表 (用于计算 percentile)
 * @returns {Object} - benchmarkEvidence 条目
 */
export function toBenchmarkEvidence(modelData, allModels) {
  // 计算 percentile
  const allRates = allModels.map(m => m.passRate).filter(rate => rate > 0)
  const atOrBelow = allRates.filter(rate => rate <= modelData.passRate).length
  const percentile = allRates.length > 0 ? atOrBelow / allRates.length : 0

  return {
    benchmark: 'deepswe',
    benchmarkVersion: 'codexradar',
    sourceModel: modelData.deepsweModel,
    domain: 'coding',
    metric: 'pass_at_1',
    rawValue: modelData.passRate,
    uncertainty: 0, // dradar 不提供 CI
    cohortPercentile: percentile,
    taskCount: modelData.cells,
    cohortSize: allModels.length,
    effort: modelData.bestEffort,
    selectionBasis: 'best_available_effort',
    sourceUrl: 'https://deng.codexradar.com/',
    capturedAt: new Date().toISOString().split('T')[0],
  }
}

/**
 * 主函数：抓取并转换 dradar 数据
 * @param {Object} modelMap - dradar 模型名 -> CCX canonicalModel 映射
 * @returns {Promise<Object>} - {canonicalModel: {benchmarkEvidence, costData, efforts}}
 */
export async function fetchDradarData(modelMap) {
  try {
    const [leaderboard, table] = await Promise.all([
      fetchLeaderboard(),
      fetchTable().catch(() => null), // table 可能不可用
    ])

    const bestPerModel = extractBestPerModel(leaderboard, modelMap)
    const costData = table ? extractCostData(table, modelMap) : {}

    const result = {}

    for (const [canonical, modelData] of Object.entries(bestPerModel)) {
      const evidence = toBenchmarkEvidence(modelData, Object.values(bestPerModel))

      if (!result[canonical]) {
        result[canonical] = {
          benchmarkEvidence: [],
          costData: {},
          efforts: {},
        }
      }

      result[canonical].benchmarkEvidence.push(evidence)
      result[canonical].costData = costData[canonical] || {}
      result[canonical].efforts = modelData.efforts
    }

    console.log(`[dradar] Extracted data for ${Object.keys(result).length} models`)
    return result
  } catch (err) {
    console.error(`[dradar] Failed to fetch data:`, err.message)
    throw err
  }
}

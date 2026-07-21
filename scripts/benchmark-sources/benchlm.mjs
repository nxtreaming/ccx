/**
 * benchlm.ai 数据抓取器
 *
 * 从 https://benchlm.ai 抓取模型对比数据。
 *
 * 数据来源：
 * - 对比页面: /compare/{modelA}-vs-{modelB}
 * - 数据嵌入在 __NEXT_DATA__ 中 (Next.js SSR)
 *
 * 数据结构：
 * - pageData: {modelA, modelB, scoreA, scoreB, diffRows, counts, scoreEvidenceA, scoreEvidenceB}
 * - diffRows: [{key, name, a, b}] - 各分类分数
 * - counts: {sharedBenchmarkCount, comparableCategoryCount, totalCategoryCount}
 */

import { fetchWithTimeout } from './http.mjs'

const BASE_URL = 'https://benchlm.ai'

/**
 * 获取对比页面数据
 * @param {string} modelASlug
 * @param {string} modelBSlug
 * @returns {Promise<Object>}
 */
export async function fetchComparison(modelASlug, modelBSlug) {
  const url = `${BASE_URL}/compare/${modelASlug}-vs-${modelBSlug}`

  console.log(`[benchlm] Fetching ${url}`)

  const resp = await fetchWithTimeout(url, {
    headers: {
      'User-Agent': 'ccx-benchmark-updater/1.0',
      Accept: 'text/html',
    },
  })

  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status} ${resp.statusText} for ${url}`)
  }

  const html = await resp.text()
  const match = html.match(/<script id="__NEXT_DATA__" type="application\/json">(.*?)<\/script>/s)

  if (!match) {
    throw new Error(`__NEXT_DATA__ not found in ${url}`)
  }

  const data = JSON.parse(match[1])
  return data.props.pageProps.pageData
}

/**
 * 从对比数据中提取模型分数
 * @param {Object} pageData - fetchComparison 的输出
 * @param {Object} modelMap - benchlm slug -> CCX canonicalModel 映射
 * @param {Object} categoryMap - benchlm 分类名 -> CCX 分类名映射
 * @returns {Object} - {canonicalModel, overallScore, categoryScores, counts, scoreEvidence}
 */
export function extractModelScore(pageData, modelMap, categoryMap) {
  const modelASlug = pageData.modelA.slug
  const modelBSlug = pageData.modelB.slug

  const canonicalA = modelMap[modelASlug]
  const canonicalB = modelMap[modelBSlug]

  const results = {}

  // 提取 modelA 数据
  if (canonicalA) {
    const categoryScores = {}
    for (const row of pageData.diffRows || []) {
      const ccxCategory = categoryMap[row.key]
      if (ccxCategory && row.a !== null && row.a !== undefined) {
        categoryScores[ccxCategory] = row.a
      }
    }

    results[canonicalA] = {
      canonicalModel: canonicalA,
      overallScore: pageData.scoreA,
      categoryScores,
      counts: pageData.counts,
      scoreEvidence: pageData.scoreEvidenceA,
      sourceUrl: `${BASE_URL}/compare/${modelASlug}-vs-${modelBSlug}`,
    }
  }

  // 提取 modelB 数据
  if (canonicalB) {
    const categoryScores = {}
    for (const row of pageData.diffRows || []) {
      const ccxCategory = categoryMap[row.key]
      if (ccxCategory && row.b !== null && row.b !== undefined) {
        categoryScores[ccxCategory] = row.b
      }
    }

    results[canonicalB] = {
      canonicalModel: canonicalB,
      overallScore: pageData.scoreB,
      categoryScores,
      counts: pageData.counts,
      scoreEvidence: pageData.scoreEvidenceB,
      sourceUrl: `${BASE_URL}/compare/${modelASlug}-vs-${modelBSlug}`,
    }
  }

  return results
}

/**
 * 生成 comparison URL
 * @param {string} slugA
 * @param {string} slugB
 * @returns {string}
 */
export function comparisonUrl(slugA, slugB) {
  return `${BASE_URL}/compare/${slugA}-vs-${slugB}`
}

/**
 * 生成 methodology URL
 * @returns {string}
 */
export function methodologyUrl() {
  return `${BASE_URL}/methodology`
}

/**
 * 预定义的对比组合
 * 根据 CCX 注册表中的 benchmarkProfiles，选择有意义的对比
 * 目标：确保每个模型至少出现在一个对比中，优先覆盖注册表已有模型
 */
export const COMPARISON_PAIRS = [
  // Claude Opus 4.8 vs 所有其他模型
  ['claude-opus-4-8', 'gpt-5-6-terra'],
  ['claude-opus-4-8', 'gpt-5-6-sol'],
  ['claude-opus-4-8', 'gpt-5-6-luna'],
  ['claude-opus-4-8', 'gpt-5-5'],
  ['claude-opus-4-8', 'claude-fable-5'],
  ['claude-opus-4-8', 'claude-sonnet-5'],
  ['claude-opus-4-8', 'claude-sonnet-4-6'],
  ['claude-opus-4-8', 'glm-5-2'],
  ['claude-opus-4-8', 'kimi-k2-7-code'],
  ['claude-opus-4-8', 'gpt-5-4'],
  // GPT-5.6 系列内部对比
  ['gpt-5-6-terra', 'gpt-5-6-sol'],
  ['gpt-5-6-terra', 'gpt-5-6-luna'],
  ['gpt-5-6-sol', 'gpt-5-6-luna'],
  ['gpt-5-6-terra', 'gpt-5-5'],
  ['gpt-5-6-sol', 'gpt-5-5'],
  ['gpt-5-6-terra', 'gpt-5-4'],
  ['gpt-5-6-sol', 'gpt-5-4'],
  ['gpt-5-6-luna', 'gpt-5-4'],
  // Claude 系列内部对比
  ['claude-fable-5', 'claude-sonnet-5'],
  ['claude-sonnet-5', 'claude-sonnet-4-6'],
  ['claude-fable-5', 'claude-sonnet-4-6'],
  ['claude-fable-5', 'gpt-5-5'],
  ['claude-fable-5', 'gpt-5-4'],
  ['claude-fable-5', 'gpt-5-6-terra'],
  ['claude-fable-5', 'gpt-5-6-sol'],
  ['claude-fable-5', 'gpt-5-6-luna'],
  // 其他模型对比
  ['gpt-5-5', 'gpt-5-4'],
  ['glm-5-2', 'kimi-k2-7-code'],
  ['glm-5-2', 'gpt-5-4'],
  ['kimi-k2-7-code', 'gpt-5-4'],
  ['glm-5-2', 'gpt-5-5'],
  ['kimi-k2-7-code', 'gpt-5-5'],
  ['gemini-3-5-flash', 'claude-haiku-4-5'],
  ['gemini-3-5-flash', 'gpt-5-4'],
  ['claude-haiku-4-5', 'gpt-5-4'],
]

/**
 * 主函数：抓取 benchlm.ai 数据
 * @param {Object} modelMap - benchlm slug -> CCX canonicalModel 映射
 * @param {Object} categoryMap - benchlm 分类名 -> CCX 分类名映射
 * @returns {Promise<Object>} - {canonicalModel: {overallScore, categoryScores, counts, scoreEvidence, sources}}
 */
export async function fetchBenchlmData(modelMap, categoryMap) {
  const result = {}
  const errors = []

  // 去重对比对（避免重复抓取）
  const uniquePairs = []
  const seen = new Set()
  for (const [a, b] of COMPARISON_PAIRS) {
    const key = [a, b].sort().join('-vs-')
    if (!seen.has(key)) {
      seen.add(key)
      uniquePairs.push([a, b])
    }
  }

  for (const [slugA, slugB] of uniquePairs) {
    try {
      const pageData = await fetchComparison(slugA, slugB)
      const scores = extractModelScore(pageData, modelMap, categoryMap)

      for (const [canonical, data] of Object.entries(scores)) {
        if (!result[canonical]) {
          result[canonical] = {
            overallScore: data.overallScore,
            categoryScores: {},
            counts: data.counts,
            scoreEvidence: data.scoreEvidence,
            sources: [],
          }
        }

        // 合并 categoryScores (取最大值，因为不同对比可能覆盖不同分类)
        for (const [cat, score] of Object.entries(data.categoryScores)) {
          if (!result[canonical].categoryScores[cat] || score > result[canonical].categoryScores[cat]) {
            result[canonical].categoryScores[cat] = score
          }
        }

        // 添加 source URL
        const sourceUrl = `${BASE_URL}/compare/${slugA}-vs-${slugB}`
        if (!result[canonical].sources.includes(sourceUrl)) {
          result[canonical].sources.push(sourceUrl)
        }
      }
    } catch (err) {
      errors.push({ pair: `${slugA}-vs-${slugB}`, error: err.message })
      console.warn(`[benchlm] Failed to fetch ${slugA}-vs-${slugB}:`, err.message)
    }
  }

  // 添加 methodology URL 到所有结果
  const methUrl = methodologyUrl()
  for (const canonical of Object.keys(result)) {
    if (!result[canonical].sources.includes(methUrl)) {
      result[canonical].sources.push(methUrl)
    }
  }

  console.log(`[benchlm] Extracted data for ${Object.keys(result).length} models`)
  if (errors.length > 0) {
    console.warn(`[benchlm] ${errors.length} comparisons failed`)
  }

  if (Object.keys(result).length === 0) {
    const detail = errors[0]?.error || 'no mapped benchmark results'
    throw new Error(`all ${uniquePairs.length} comparisons failed: ${detail}`)
  }

  return result
}

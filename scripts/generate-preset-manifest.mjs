import { createHash } from 'node:crypto'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const docsOutputDir = join(root, 'docs/public/presets')
const embeddedOutputDir = join(root, 'backend-go/internal/presetstore/embedded')

const SHARDS = [
  {
    fileName: 'subscription-preset.json',
    kind: 'subscriptionPreset',
    sourcePath: join(root, 'shared/subscription-preset/subscription-preset.json'),
    transform: value => validateSchemaVersion(value, 'shared/subscription-preset/subscription-preset.json'),
  },
  {
    fileName: 'model-registry.json',
    kind: 'modelRegistry',
    sourcePath: join(root, 'shared/model-registry/ccx_model_registry.json'),
    transform: value => validateSchemaVersion(value, 'shared/model-registry/ccx_model_registry.json'),
  },
  {
    fileName: 'channel-presets.json',
    kind: 'channelPresets',
    transform: () => buildChannelPresetsShard(),
  },
  {
    fileName: 'builtin-manifest.json',
    kind: 'builtinManifest',
    sourcePath: join(root, 'shared/builtin-models-manifest/builtin-models-manifest.json'),
    transform: value => validateBuiltinManifest(value, 'shared/builtin-models-manifest/builtin-models-manifest.json'),
  },
]

const CHANNEL_PRESET_SOURCES = {
  claudeMessages: join(root, 'shared/channel-presets/claude-messages.json'),
  openAIChat: join(root, 'shared/channel-presets/openai-chat.json'),
  codexResponses: join(root, 'shared/channel-presets/codex-responses.json'),
  openAIMessages: join(root, 'shared/channel-presets/openai-messages.json'),
}

function main() {
  mkdirSync(docsOutputDir, { recursive: true })
  mkdirSync(embeddedOutputDir, { recursive: true })

  const renderedShards = SHARDS.map(renderShard)
  const index = buildIndex(renderedShards)
  const indexContent = formatJson(index)

  writeArtifact(docsOutputDir, 'index.json', indexContent)
  writeArtifact(embeddedOutputDir, 'index.json', indexContent)

  for (const shard of renderedShards) {
    writeArtifact(docsOutputDir, shard.fileName, shard.content)
    writeArtifact(embeddedOutputDir, shard.fileName, shard.content)
  }
}

function renderShard(shard) {
  const value = shard.sourcePath
    ? shard.transform(readJsonFile(shard.sourcePath))
    : shard.transform()
  const content = shard.sourcePath ? formatLikeSource(shard.sourcePath, value) : formatJson(value)
  const sha256 = sha256Hex(content)
  return {
    kind: shard.kind,
    fileName: shard.fileName,
    content,
    sha256,
    sizeBytes: Buffer.byteLength(content),
  }
}

function buildChannelPresetsShard() {
  const result = { schemaVersion: 1 }
  for (const [key, sourcePath] of Object.entries(CHANNEL_PRESET_SOURCES)) {
    const value = validateSchemaVersion(readJsonFile(sourcePath), relativeToRoot(sourcePath))
    result[key] = value
  }
  return result
}

function buildIndex(shards) {
  const dataVersion = resolveDataVersion()
  const publishedAt = resolvePublishedAt()
  const index = {
    schemaVersion: 1,
    dataVersion,
    shards: shards.map(shard => ({
      kind: shard.kind,
      url: `./${shard.fileName}`,
      sha256: shard.sha256,
      sizeBytes: shard.sizeBytes,
    })),
  }
  if (publishedAt) {
    index.publishedAt = publishedAt
  }
  return index
}

function resolveDataVersion() {
  const explicit = process.env.PRESET_DATA_VERSION?.trim()
  if (explicit) {
    return explicit
  }
  const version = readFileSync(join(root, 'VERSION'), 'utf8').trim()
  if (!version) {
    throw new Error('VERSION must not be empty')
  }
  const publishedDate = resolveStableDate()
  return `${version}+${publishedDate}`
}

function resolvePublishedAt() {
  const epochText = process.env.SOURCE_DATE_EPOCH?.trim()
  if (!epochText) {
    return undefined
  }
  if (!/^\d+$/.test(epochText)) {
    throw new Error('SOURCE_DATE_EPOCH must be an integer number of seconds')
  }
  const epoch = Number(epochText)
  if (!Number.isSafeInteger(epoch) || epoch < 0) {
    throw new Error('SOURCE_DATE_EPOCH must be a non-negative safe integer')
  }
  return new Date(epoch * 1000).toISOString()
}

function resolveStableDate() {
  const epochText = process.env.SOURCE_DATE_EPOCH?.trim()
  if (epochText) {
    if (!/^\d+$/.test(epochText)) {
      throw new Error('SOURCE_DATE_EPOCH must be an integer number of seconds')
    }
    const epoch = Number(epochText)
    if (!Number.isSafeInteger(epoch) || epoch < 0) {
      throw new Error('SOURCE_DATE_EPOCH must be a non-negative safe integer')
    }
    return new Date(epoch * 1000).toISOString().slice(0, 10).replace(/-/g, '')
  }
  return new Date().toISOString().slice(0, 10).replace(/-/g, '')
}

function readJsonFile(path) {
  const raw = readFileSync(path, 'utf8')
  try {
    return JSON.parse(raw)
  } catch (error) {
    throw new Error(`Invalid JSON in ${relativeToRoot(path)}: ${error.message}`)
  }
}

function validateSchemaVersion(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must contain a JSON object`)
  }
  if (value.schemaVersion !== 1) {
    throw new Error(`${label} schemaVersion must be 1`)
  }
  return value
}

function validateBuiltinManifest(value, label) {
  const manifest = validateSchemaVersion(value, label)
  if (!Array.isArray(manifest.manifests) || manifest.manifests.length === 0) {
    throw new Error(`${label} manifests must be a non-empty array`)
  }
  for (const [index, entry] of manifest.manifests.entries()) {
    if (!entry || typeof entry !== 'object' || Array.isArray(entry)) {
      throw new Error(`${label} manifests[${index}] must be an object`)
    }
    if (typeof entry.baseUrlPattern !== 'string' || entry.baseUrlPattern.trim() === '') {
      throw new Error(`${label} manifests[${index}].baseUrlPattern must be a non-empty string`)
    }
    if (typeof entry.serviceType !== 'string' || entry.serviceType.trim() === '') {
      throw new Error(`${label} manifests[${index}].serviceType must be a non-empty string`)
    }
    if (entry.modelsUrl !== undefined) {
      if (typeof entry.modelsUrl !== 'string' || !/^https:\/\//.test(entry.modelsUrl)) {
        throw new Error(`${label} manifests[${index}].modelsUrl must be an https URL when present`)
      }
    }
    if (!Array.isArray(entry.modelIds) || entry.modelIds.length === 0 || entry.modelIds.some(model => typeof model !== 'string' || model.trim() === '')) {
      throw new Error(`${label} manifests[${index}].modelIds must be a non-empty string array`)
    }
    if (entry.excludeModelPatterns !== undefined) {
      if (!Array.isArray(entry.excludeModelPatterns) || entry.excludeModelPatterns.some(pattern => typeof pattern !== 'string' || pattern.trim() === '')) {
        throw new Error(`${label} manifests[${index}].excludeModelPatterns must be a non-empty string array when present`)
      }
      for (const [patternIndex, pattern] of entry.excludeModelPatterns.entries()) {
        try {
          new RegExp(pattern)
        } catch (error) {
          throw new Error(`${label} manifests[${index}].excludeModelPatterns[${patternIndex}] is invalid: ${error.message}`)
        }
      }
    }
    if (typeof entry.disableProbe !== 'boolean') {
      throw new Error(`${label} manifests[${index}].disableProbe must be a boolean`)
    }
  }
  return manifest
}

function formatJson(value) {
  return `${JSON.stringify(value, null, 2)}\n`
}

function formatLikeSource(sourcePath, value) {
  if (!sourcePath) {
    return formatJson(value)
  }
  const sourceText = readFileSync(sourcePath, 'utf8')
  const sourceValue = JSON.parse(sourceText)
  if (JSON.stringify(sourceValue) === JSON.stringify(value)) {
    return ensureTrailingNewline(sourceText)
  }
  return formatJson(value)
}

function ensureTrailingNewline(text) {
  return text.endsWith('\n') ? text : `${text}\n`
}

function sha256Hex(content) {
  return createHash('sha256').update(content).digest('hex')
}

function writeArtifact(dir, fileName, content) {
  writeFileSync(join(dir, fileName), content)
  const checksum = `${sha256Hex(content)}  ${fileName}\n`
  writeFileSync(join(dir, `${fileName}.sha256`), checksum)
}

function relativeToRoot(path) {
  return path.startsWith(root) ? path.slice(root.length + 1) : path
}

main()

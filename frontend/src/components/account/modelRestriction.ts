export type ModelRestrictionMode = 'whitelist' | 'mapping'

export interface ModelMappingRow {
  from: string
  to: string
}

export function parseWhitelistPaste(input: string): string[] {
  const tokens = input.split(/[\s,]+/)
  const seen = new Set<string>()
  const result: string[] = []
  for (const token of tokens) {
    const trimmed = token.trim()
    if (!trimmed || trimmed === '*') {
      continue
    }
    if (!seen.has(trimmed)) {
      seen.add(trimmed)
      result.push(trimmed)
    }
  }
  return result
}

export function parseMappingPaste(input: string): ModelMappingRow[] {
  const lines = input.split(/\r?\n/)
  const order: string[] = []
  const mapping = new Map<string, string>()
  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) {
      continue
    }
    let separator = ''
    let index = -1
    if (line.includes('->')) {
      separator = '->'
      index = line.indexOf('->')
    } else if (line.includes(',')) {
      separator = ','
      index = line.indexOf(',')
    } else if (line.includes('\t')) {
      separator = '\t'
      index = line.indexOf('\t')
    }
    if (index < 0) {
      continue
    }
    const from = line.slice(0, index).trim()
    const to = line.slice(index + separator.length).trim()
    if (!from || !to) {
      continue
    }
    if (mapping.has(from)) {
      const pos = order.indexOf(from)
      if (pos >= 0) {
        order.splice(pos, 1)
      }
    }
    mapping.set(from, to)
    order.push(from)
  }
  return order.map((from) => ({ from, to: mapping.get(from) ?? '' })).filter((row) => row.to !== '')
}

export function decodeModelRestriction(mapping?: Record<string, string> | null): {
  mode: ModelRestrictionMode
  allowedModels: string[]
  modelMappings: ModelMappingRow[]
} {
  if (!mapping || typeof mapping !== 'object' || Object.keys(mapping).length === 0) {
    return {
      mode: 'whitelist',
      allowedModels: [],
      modelMappings: []
    }
  }
  const entries = Object.entries(mapping)
  const isWhitelist = entries.every(([from, to]) => from === to)
  if (isWhitelist) {
    const allowedModels = entries
      .map(([from]) => from.trim())
      .filter((model) => model.length > 0 && model !== '*')
    return {
      mode: 'whitelist',
      allowedModels,
      modelMappings: []
    }
  }
  const modelMappings = entries
    .map(([from, to]) => ({ from: from.trim(), to: to.trim() }))
    .filter((row) => row.from.length > 0 && row.to.length > 0)
  return {
    mode: 'mapping',
    allowedModels: [],
    modelMappings
  }
}

export function summarizeModelRestriction(mapping?: Record<string, string> | null): string {
  const decoded = decodeModelRestriction(mapping)
  if (decoded.mode === 'whitelist') {
    const models = decoded.allowedModels.slice().sort((a, b) => a.localeCompare(b))
    if (models.length === 0) {
      return '全部模型'
    }
    const visible = models.slice(0, 3)
    const remaining = models.length - visible.length
    return `白名单: ${visible.join(', ')}${remaining > 0 ? ` (+${remaining})` : ''}`
  }
  const mappings = decoded.modelMappings
    .slice()
    .sort((a, b) => a.from.localeCompare(b.from))
  if (mappings.length === 0) {
    return '全部模型'
  }
  const visible = mappings.slice(0, 3).map((row) => `${row.from} -> ${row.to}`)
  const remaining = mappings.length - visible.length
  return `映射: ${visible.join('; ')}${remaining > 0 ? ` (+${remaining})` : ''}`
}

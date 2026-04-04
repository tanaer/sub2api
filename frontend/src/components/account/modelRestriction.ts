export type ModelRestrictionMode = 'whitelist' | 'mapping'

export interface ModelMappingRow {
  from: string
  to: string
}

export interface ModelRestrictionSummaryOptions {
  labels?: {
    allModels?: string
    whitelist?: string
    mapping?: string
  }
  formatMore?: (remaining: number) => string
  mappingSeparator?: string
  mappingArrow?: string
}

function isValidWildcardPattern(pattern: string): boolean {
  const starIndex = pattern.indexOf('*')
  if (starIndex === -1) return true
  return starIndex === pattern.length - 1 && pattern.lastIndexOf('*') === starIndex
}

export function parseWhitelistPaste(input: string): string[] {
  const tokens = input.split(/[\s,]+/)
  const seen = new Set<string>()
  const result: string[] = []
  for (const token of tokens) {
    const trimmed = token.trim()
    if (!trimmed || trimmed.includes('*')) {
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
    if (!isValidWildcardPattern(from) || to.includes('*')) {
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
  const isWhitelist = entries.every(([from, to]) => from === to && !from.includes('*'))
  if (isWhitelist) {
    const allowedModels = entries
      .map(([from]) => from.trim())
      .filter((model) => model.length > 0 && !model.includes('*'))
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

export function exportWhitelist(models: string[]): string {
  return models.filter((m) => m.trim()).join('\n')
}

export function exportMapping(mappings: ModelMappingRow[]): string {
  return mappings
    .filter((row) => row.from.trim() && row.to.trim())
    .map((row) => `${row.from} -> ${row.to}`)
    .join('\n')
}

export function summarizeModelRestriction(
  mapping?: Record<string, string> | null,
  options?: ModelRestrictionSummaryOptions
): string {
  const labels = {
    allModels: options?.labels?.allModels ?? 'All models',
    whitelist: options?.labels?.whitelist ?? 'Whitelist',
    mapping: options?.labels?.mapping ?? 'Mapping'
  }
  const formatMore = options?.formatMore ?? ((remaining) => ` (+${remaining})`)
  const mappingSeparator = options?.mappingSeparator ?? '; '
  const mappingArrow = options?.mappingArrow ?? ' -> '
  const decoded = decodeModelRestriction(mapping)
  if (decoded.mode === 'whitelist') {
    const models = decoded.allowedModels.slice().sort((a, b) => a.localeCompare(b))
    if (models.length === 0) {
      return labels.allModels
    }
    const visible = models.slice(0, 3)
    const remaining = models.length - visible.length
    return `${labels.whitelist}: ${visible.join(', ')}${remaining > 0 ? formatMore(remaining) : ''}`
  }
  const mappings = decoded.modelMappings
    .slice()
    .sort((a, b) => a.from.localeCompare(b.from))
  if (mappings.length === 0) {
    return labels.allModels
  }
  const visible = mappings.slice(0, 3).map((row) => `${row.from}${mappingArrow}${row.to}`)
  const remaining = mappings.length - visible.length
  return `${labels.mapping}: ${visible.join(mappingSeparator)}${remaining > 0 ? formatMore(remaining) : ''}`
}

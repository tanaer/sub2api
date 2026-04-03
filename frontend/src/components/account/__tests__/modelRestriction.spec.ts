import { describe, it, expect } from 'vitest'
import {
  parseWhitelistPaste,
  parseMappingPaste,
  decodeModelRestriction,
  summarizeModelRestriction
} from '../modelRestriction'

describe('modelRestriction helpers', () => {
  it('白名单粘贴支持空格/逗号/换行混合输入', () => {
    const input = 'gpt-4, claude-3\n gemini-2.5-pro\tglm-4.5-air  '
    expect(parseWhitelistPaste(input)).toEqual([
      'gpt-4',
      'claude-3',
      'gemini-2.5-pro',
      'glm-4.5-air'
    ])
  })

  it('白名单粘贴去重并忽略通配符条目', () => {
    const input = 'gpt-4, gpt-4, claude-*, *, gemini-2.5-pro'
    expect(parseWhitelistPaste(input)).toEqual(['gpt-4', 'gemini-2.5-pro'])
  })

  it('映射粘贴支持 -> / 逗号 / TAB', () => {
    const input = [
      'sonnet -> glm-4.5-air',
      'claude-3,claude-3-haiku',
      'foo\tbar'
    ].join('\n')
    expect(parseMappingPaste(input)).toEqual([
      { from: 'sonnet', to: 'glm-4.5-air' },
      { from: 'claude-3', to: 'claude-3-haiku' },
      { from: 'foo', to: 'bar' }
    ])
  })

  it('映射粘贴遵循通配符格式与目标模型规则', () => {
    const input = [
      'claude-* -> claude-3',
      'bad*pattern -> glm-4.5-air',
      'sonnet -> glm-*',
      '* -> glm-4.5-air'
    ].join('\n')
    expect(parseMappingPaste(input)).toEqual([
      { from: 'claude-*', to: 'claude-3' },
      { from: '*', to: 'glm-4.5-air' }
    ])
  })

  it('映射粘贴重复 from 取最后一次', () => {
    const input = [
      'sonnet -> glm-4.5-air',
      'haiku -> claude-3-haiku',
      'sonnet -> glm-4.5-flash'
    ].join('\n')
    expect(parseMappingPaste(input)).toEqual([
      { from: 'haiku', to: 'claude-3-haiku' },
      { from: 'sonnet', to: 'glm-4.5-flash' }
    ])
  })

  it('摘要最多展示 3 项并附 (+N)', () => {
    const whitelistMapping = {
      b: 'b',
      a: 'a',
      c: 'c',
      d: 'd',
      e: 'e'
    }
    expect(summarizeModelRestriction(whitelistMapping)).toBe('Whitelist: a, b, c (+2)')

    const mapping = {
      b: 'x',
      a: 'y',
      c: 'z',
      d: 'w'
    }
    expect(summarizeModelRestriction(mapping)).toBe('Mapping: a -> y; b -> x; c -> z (+1)')
  })

  it('摘要空值返回全部模型文案', () => {
    expect(summarizeModelRestriction(undefined)).toBe('All models')
    expect(summarizeModelRestriction({})).toBe('All models')
  })

  it('摘要支持自定义标签', () => {
    const labels = {
      allModels: 'All',
      whitelist: 'Allow',
      mapping: 'Route'
    }
    const mapping = { a: 'a', b: 'b' }
    expect(summarizeModelRestriction(mapping, { labels })).toBe('Allow: a, b')
  })

  it('model_mapping 全部 from===to 时视为白名单语义', () => {
    const decoded = decodeModelRestriction({ 'glm-4.5-air': 'glm-4.5-air' })
    expect(decoded.mode).toBe('whitelist')
    expect(decoded.allowedModels).toEqual(['glm-4.5-air'])
    expect(decoded.modelMappings).toEqual([])
  })

  it('decodeModelRestriction 处理空值与空对象', () => {
    expect(decodeModelRestriction(undefined)).toEqual({
      mode: 'whitelist',
      allowedModels: [],
      modelMappings: []
    })
    expect(decodeModelRestriction({})).toEqual({
      mode: 'whitelist',
      allowedModels: [],
      modelMappings: []
    })
  })
})

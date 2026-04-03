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

  it('白名单忽略 *，映射允许 from 带 *', () => {
    expect(parseWhitelistPaste('*, gpt-4')).toEqual(['gpt-4'])
    expect(parseMappingPaste('* -> glm-4.5-air')).toEqual([
      { from: '*', to: 'glm-4.5-air' }
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
    expect(summarizeModelRestriction(whitelistMapping)).toBe('白名单: a, b, c (+2)')

    const mapping = {
      b: 'x',
      a: 'y',
      c: 'z',
      d: 'w'
    }
    expect(summarizeModelRestriction(mapping)).toBe('映射: a -> y; b -> x; c -> z (+1)')
  })

  it('model_mapping 全部 from===to 时视为白名单语义', () => {
    const decoded = decodeModelRestriction({ 'glm-4.5-air': 'glm-4.5-air' })
    expect(decoded.mode).toBe('whitelist')
    expect(decoded.allowedModels).toEqual(['glm-4.5-air'])
    expect(decoded.modelMappings).toEqual([])
  })
})

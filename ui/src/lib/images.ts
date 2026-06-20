// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ChatImage } from '@/hooks/useSessionChat'

export function extractImagesFromValue(value: unknown): ChatImage[] {
  const found: Array<{ mediaType: string; data: string }> = []

  const walk = (node: unknown) => {
    if (!node || typeof node !== 'object') return
    if (Array.isArray(node)) {
      node.forEach(walk)
      return
    }
    const obj = node as Record<string, unknown>
    if (obj.type === 'image') {
      const direct = obj.data
      if (typeof direct === 'string' && direct) {
        const mediaType =
          (typeof obj.mimeType === 'string' && obj.mimeType) ||
          (typeof obj.media_type === 'string' && obj.media_type) ||
          'image/png'
        found.push({ mediaType, data: direct })
      } else {
        const source = obj.source
        if (source && typeof source === 'object') {
          const src = source as Record<string, unknown>
          const data = src.data
          const url = src.url
          const mediaType =
            (typeof src.media_type === 'string' && src.media_type) ||
            (typeof src.mediaType === 'string' && src.mediaType) ||
            'image/png'
          if (typeof data === 'string' && data) {
            found.push({ mediaType, data })
          } else if (typeof url === 'string' && url) {
            found.push({ mediaType, data: url })
          }
        }
      }
    }
    Object.values(obj).forEach(walk)
  }

  walk(value)
  return found.map((img, index) => ({
    id: `tool-image-${index}`,
    mediaType: img.mediaType,
    data: img.data,
  }))
}

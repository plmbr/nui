// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ReactElement, ReactNode } from 'react'

/** Flatten react-markdown code/pre children into plain text. */
export function getReactNodeText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node)
  }

  if (Array.isArray(node)) {
    return node.map(getReactNodeText).join('')
  }

  if (node && typeof node === 'object' && 'props' in node) {
    const element = node as ReactElement<{ children?: ReactNode }>
    return getReactNodeText(element.props.children)
  }

  return ''
}

export function getCodeBlockInfo(node: ReactNode): { text: string; className?: string } {
  if (!node || typeof node !== 'object' || !('props' in node)) {
    return { text: getReactNodeText(node) }
  }

  const element = node as ReactElement<{ className?: string; children?: ReactNode }>
  return {
    text: getReactNodeText(element.props.children),
    className: element.props.className,
  }
}

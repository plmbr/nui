// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useRef, useState } from 'react'

import { prepareVisualizationHtml } from '@/lib/prepareVisualizationHtml'

interface VisualizationFrameProps {
  html: string
  title?: string
}

const RESIZE_MESSAGE = 'loop-viz-resize'

function injectAutoResize(html: string): string {
  const script = `<script>
(function () {
  function report() {
    var h = Math.max(
      document.documentElement.scrollHeight,
      document.body ? document.body.scrollHeight : 0
    );
    parent.postMessage({ type: '${RESIZE_MESSAGE}', height: h }, '*');
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', report);
  } else {
    report();
  }
  window.addEventListener('load', report);
  if (typeof ResizeObserver !== 'undefined') {
    new ResizeObserver(report).observe(document.documentElement);
  }
})();
</script>`

  const lower = html.toLowerCase()
  const bodyClose = lower.lastIndexOf('</body>')
  if (bodyClose >= 0) {
    return html.slice(0, bodyClose) + script + html.slice(bodyClose)
  }
  return html + script
}

export function VisualizationFrame({ html, title }: VisualizationFrameProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [height, setHeight] = useState<number>()
  const chartScriptSrc =
    typeof window !== 'undefined'
      ? `${window.location.origin}/vendor/chart.min.js`
      : '/vendor/chart.min.js'
  const srcDoc = useMemo(
    () => injectAutoResize(prepareVisualizationHtml(html, chartScriptSrc)),
    [html, chartScriptSrc],
  )

  useEffect(() => {
    const iframe = iframeRef.current
    if (!iframe) return

    const measure = () => {
      try {
        const doc = iframe.contentDocument
        if (!doc?.documentElement) return
        const next = Math.max(doc.documentElement.scrollHeight, doc.body?.scrollHeight ?? 0)
        if (next > 0) setHeight(next)
      } catch {
        // ignore
      }
    }

    const onMessage = (event: MessageEvent) => {
      if (event.source !== iframe.contentWindow) return
      const data = event.data as { type?: string; height?: number }
      if (data?.type !== RESIZE_MESSAGE || typeof data.height !== 'number') return
      if (data.height > 0) setHeight(Math.ceil(data.height))
    }

    window.addEventListener('message', onMessage)
    iframe.addEventListener('load', measure)
    return () => {
      window.removeEventListener('message', onMessage)
      iframe.removeEventListener('load', measure)
    }
  }, [srcDoc])

  return (
    <iframe
      ref={iframeRef}
      srcDoc={srcDoc}
      title={title || 'Chart'}
      sandbox="allow-scripts allow-same-origin"
      scrolling="no"
      className="visualization-frame"
      style={height ? { height: `${height}px` } : undefined}
    />
  )
}

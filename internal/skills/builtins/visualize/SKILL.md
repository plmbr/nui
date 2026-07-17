---
name: visualize
description: Render charts inline in nui chat via show_visualization — never use dataviz Skill or Write
---

# Visualize (nui)

When the user asks for a chart, graph, table, or dashboard, render it **inline in the nui chat panel** using **`show_visualization`** on the **`nui-viz`** MCP server.

## Required workflow (same turn)

1. Build self-contained HTML (Chart.js v3/v4 API from CDNs is fine — nui rewrites CDN chart.js tags to its bundled v4 copy at `/vendor/chart.min.js`).
2. Call **`show_visualization`** with the HTML in the **`html`** field.
3. Optionally set **`title`**.

Do this **before** sending any closing assistant text. Never stop after "building the chart" without calling the tool.

Do **not** paste markdown images, `data:image/...` URIs, or base64 in your text after calling **show_visualization** — nui already renders the chart inline.

## Do not

- **Do not** invoke the **Skill** tool or the **dataviz** bundled skill.
- **Do not** use **Write** / **Edit** to save `.html` files.
- **Do not** tell the user to open a file in a browser.

## Example

```json
{
  "title": "California demographics",
  "html": "<!DOCTYPE html><html><head><script src=\"https://cdn.jsdelivr.net/npm/chart.js\"></script></head><body><canvas id=\"c\"></canvas><script>new Chart(document.getElementById('c'),{type:'pie',data:{labels:['A','B'],datasets:[{data:[60,40]}]}});</script></body></html>"
}
```

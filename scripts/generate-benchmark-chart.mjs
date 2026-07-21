#!/usr/bin/env node

import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const DEFAULT_INPUT = '/tmp/benchmark-viz-data.json'
const DEFAULT_OUTPUT = '/tmp/benchmark-chart.html'

function optionValue(args, name, fallback) {
  const inline = args.find(arg => arg.startsWith(`${name}=`))
  if (inline) return inline.slice(name.length + 1)
  const index = args.indexOf(name)
  return index >= 0 && args[index + 1] ? args[index + 1] : fallback
}

function validateRows(raw) {
  if (!raw || !Array.isArray(raw.data)) throw new Error('输入文件缺少 data 数组')
  const rows = raw.data.filter(row => (
    row && typeof row.model === 'string' && typeof row.source === 'string'
      && Number.isFinite(row.pass_rate)
      && (Number.isFinite(row.mean_cost) || Number.isFinite(row.median_cost))
  ))
  if (rows.length === 0) throw new Error('输入文件没有可绘制的 benchmark 数据')
  return rows
}

export function renderBenchmarkChart(rows) {
  const serializedRows = JSON.stringify(rows).replaceAll('<', '\\u003c')
  return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>模型能力-成本边界</title>
<style>
:root {
  color-scheme: light dark;
  --background: #f7f8f6;
  --surface: #ffffff;
  --foreground: #151714;
  --muted: #62675f;
  --border: #d8ddd5;
  --grid: #e7eae5;
  --accent: #176a4b;
  --accent-soft: #dceee5;
  --frontier: #111713;
  --series-1: #1769aa;
  --series-2: #b44b27;
  --series-3: #168269;
  --series-4: #8a50a5;
  --series-5: #c17b08;
  --series-6: #c33b56;
  --series-7: #417c24;
  --series-8: #4f5fc4;
  --series-9: #9b5c18;
  --series-10: #287a86;
}
@media (prefers-color-scheme: dark) {
  :root {
    --background: #171917;
    --surface: #202320;
    --foreground: #f1f3ef;
    --muted: #aeb5aa;
    --border: #3b403a;
    --grid: #2c302c;
    --accent: #58c899;
    --accent-soft: #233d31;
    --frontier: #f4f6f2;
    --series-1: #62a9e4;
    --series-2: #ef845f;
    --series-3: #4fc2a5;
    --series-4: #c58cdb;
    --series-5: #e6ae43;
    --series-6: #ed7890;
    --series-7: #86be65;
    --series-8: #8996ea;
    --series-9: #d69a57;
    --series-10: #65b6c0;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--background);
  color: var(--foreground);
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  letter-spacing: 0;
}
.page { width: min(1180px, 100%); margin: 0 auto; padding: 24px; }
header { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
h1 { margin: 0; font-size: 22px; font-weight: 500; }
#summary { color: var(--muted); font-size: 12px; text-align: right; }
.controls { display: flex; align-items: end; flex-wrap: wrap; gap: 14px 20px; padding: 12px 0; border-block: 1px solid var(--border); }
.control { display: grid; gap: 6px; }
.control-label { color: var(--muted); font-size: 11px; font-weight: 500; }
.segmented { display: inline-flex; padding: 2px; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); }
.segmented button { min-height: 30px; padding: 4px 11px; border: 0; border-radius: 4px; color: var(--muted); background: transparent; font: inherit; cursor: pointer; }
.segmented button[aria-pressed="true"] { color: var(--foreground); background: var(--accent-soft); }
select { min-height: 36px; padding: 5px 30px 5px 10px; border: 1px solid var(--border); border-radius: 6px; color: var(--foreground); background: var(--surface); font: inherit; }
button:focus-visible, select:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.chart-shell { position: relative; width: 100%; margin-top: 18px; }
.benchmark-chart { display: block; width: 100%; color: var(--foreground); }
.benchmark-chart .grid-line { stroke: var(--grid); stroke-width: 1; vector-effect: non-scaling-stroke; }
.benchmark-chart .axis-line { stroke: var(--border); stroke-width: 1; vector-effect: non-scaling-stroke; }
.benchmark-chart .axis-text, .benchmark-chart .axis-title { fill: var(--muted); font-size: 11px; }
.benchmark-chart .axis-title { font-size: 12px; }
.benchmark-chart .trajectory { fill: none; stroke-width: 1.5; opacity: .42; vector-effect: non-scaling-stroke; transition: d 180ms ease; }
.benchmark-chart .frontier { fill: none; stroke: var(--frontier); stroke-width: 2.5; vector-effect: non-scaling-stroke; transition: d 180ms ease; }
.benchmark-chart .point { stroke: var(--surface); stroke-width: 1.5; cursor: crosshair; vector-effect: non-scaling-stroke; transition: cx 180ms ease, cy 180ms ease; }
.benchmark-chart .point.is-pareto { stroke: var(--frontier); stroke-width: 2; }
.benchmark-chart .label { fill: var(--foreground); font-size: 11px; font-weight: 500; }
.benchmark-chart .label-link { stroke: var(--border); stroke-width: 1; vector-effect: non-scaling-stroke; }
.tooltip { position: absolute; z-index: 2; width: max-content; max-width: min(260px, calc(100% - 16px)); padding: 9px 11px; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--foreground); box-shadow: 0 8px 22px rgb(0 0 0 / .14); pointer-events: none; opacity: 0; transform: translateY(3px); transition: opacity 100ms ease, transform 100ms ease; }
.tooltip.visible { opacity: 1; transform: translateY(0); }
.tooltip strong { display: block; margin-bottom: 4px; font-weight: 500; }
.tooltip-line { color: var(--muted); font-size: 12px; }
.legend { display: flex; flex-wrap: wrap; gap: 7px 15px; min-height: 24px; margin: 4px 0 20px; color: var(--muted); font-size: 11px; }
.legend-item { display: inline-flex; align-items: center; gap: 6px; }
.legend-line { width: 17px; height: 3px; border-radius: 2px; }
.table-section { border-top: 1px solid var(--border); padding-top: 14px; }
.table-heading { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; margin-bottom: 8px; }
h2 { margin: 0; font-size: 15px; font-weight: 500; }
#table-count { color: var(--muted); font-size: 11px; }
table { width: 100%; border-collapse: collapse; table-layout: fixed; font-size: 12px; }
th, td { padding: 8px 9px; border-bottom: 1px solid var(--grid); text-align: left; vertical-align: middle; overflow-wrap: anywhere; }
th { color: var(--muted); font-size: 11px; font-weight: 500; }
tbody tr:hover { background: var(--accent-soft); }
.numeric { text-align: right; font-variant-numeric: tabular-nums; }
.pareto-mark { color: var(--accent); font-weight: 500; }
.col-model { width: 28%; }
.col-effort { width: 12%; }
.col-rate, .col-cost { width: 15%; }
.col-pareto { width: 12%; }
.col-source { width: 18%; }
@media (prefers-reduced-motion: reduce) { * { transition-duration: 0s !important; } }
@media (max-width: 640px) {
  .page { padding: 16px 12px; }
  header { display: block; }
  #summary { margin-top: 6px; text-align: left; }
  .controls { align-items: stretch; gap: 12px; }
  .control { flex: 1 1 140px; }
  .segmented { display: flex; }
  .segmented button { flex: 1; }
  select { width: 100%; }
  th, td { padding-inline: 5px; }
  .col-model { width: 38%; }
  .col-effort { width: 18%; }
  .col-rate, .col-cost { width: 22%; }
  .col-pareto, .col-source { display: none; }
}
</style>
</head>
<body>
<main class="page">
  <header>
    <h1>模型能力-成本边界</h1>
    <div id="summary" aria-live="polite"></div>
  </header>
  <div class="controls" aria-label="图表筛选">
    <div class="control">
      <span class="control-label">成本口径</span>
      <div class="segmented" id="metric-control">
        <button type="button" data-value="mean_cost" aria-pressed="true">平均成本</button>
        <button type="button" data-value="median_cost" aria-pressed="false">中位成本</button>
      </div>
    </div>
    <div class="control">
      <span class="control-label">成本范围</span>
      <div class="segmented" id="range-control">
        <button type="button" data-value="focus" aria-pressed="true">聚焦范围</button>
        <button type="button" data-value="full" aria-pressed="false">全部成本</button>
      </div>
    </div>
    <label class="control" for="source-control">
      <span class="control-label">数据源</span>
      <select id="source-control"></select>
    </label>
  </div>
  <section class="chart-shell" id="chart-shell">
    <svg class="benchmark-chart" id="chart" role="img" aria-labelledby="chart-title chart-description">
      <title id="chart-title">模型 pass@1 与单任务成本散点图</title>
      <desc id="chart-description">模型思考强度点由轨迹连接，深色折线表示成本与能力的 Pareto 前沿。</desc>
      <defs><clipPath id="plot-clip"><rect id="clip-rect"></rect></clipPath></defs>
      <g id="grid"></g>
      <g id="axes"></g>
      <g id="trajectories" clip-path="url(#plot-clip)"></g>
      <path id="frontier" class="frontier" clip-path="url(#plot-clip)"></path>
      <g id="points" clip-path="url(#plot-clip)"></g>
      <g id="labels"></g>
    </svg>
    <div class="tooltip" id="tooltip" role="status"></div>
  </section>
  <div class="legend" id="legend" aria-label="模型图例"></div>
  <section class="table-section">
    <div class="table-heading"><h2>当前范围</h2><span id="table-count"></span></div>
    <table>
      <thead><tr>
        <th class="col-model">模型</th><th class="col-effort">强度</th>
        <th class="col-rate numeric">pass@1</th><th class="col-cost numeric" id="cost-heading">平均成本</th>
        <th class="col-pareto">边界</th><th class="col-source">数据源</th>
      </tr></thead>
      <tbody id="table-body"></tbody>
    </table>
  </section>
</main>
<script>
const RAW_ROWS = ${serializedRows};
const state = { metric: 'mean_cost', range: 'focus', source: 'all' };
const effortRank = new Map([['low', 0], ['medium', 1], ['high', 2], ['xhigh', 3], ['max', 4]]);
const palette = Array.from({ length: 10 }, (_, index) => 'var(--series-' + (index + 1) + ')');
const modelNames = [...new Set(RAW_ROWS.map(row => row.model))].sort();
const colors = new Map(modelNames.map((model, index) => [model, palette[index % palette.length]]));
const svg = document.getElementById('chart');
const shell = document.getElementById('chart-shell');
const tooltip = document.getElementById('tooltip');
const ns = 'http://www.w3.org/2000/svg';
let geometry = null;

function svgNode(name, attributes, text) {
  const node = document.createElementNS(ns, name);
  Object.entries(attributes || {}).forEach(([key, value]) => node.setAttribute(key, String(value)));
  if (text != null) node.textContent = text;
  return node;
}

function quantile(values, q) {
  const sorted = [...values].sort((a, b) => a - b);
  if (!sorted.length) return 0;
  const position = (sorted.length - 1) * q;
  const lower = Math.floor(position);
  const fraction = position - lower;
  return sorted[lower + 1] == null ? sorted[lower] : sorted[lower] + fraction * (sorted[lower + 1] - sorted[lower]);
}

function paretoFrontier(rows) {
  const sorted = [...rows].sort((a, b) => a.cost - b.cost || b.pass_rate - a.pass_rate);
  const frontier = [];
  let bestRate = -Infinity;
  sorted.forEach(row => {
    if (row.pass_rate > bestRate + 1e-9) {
      frontier.push(row);
      bestRate = row.pass_rate;
    }
  });
  return frontier;
}

function niceMax(value) {
  if (value <= 0) return 1;
  const magnitude = 10 ** Math.floor(Math.log10(value));
  const normalized = value / magnitude;
  const rounded = [1, 1.2, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10].find(candidate => normalized <= candidate) || 10;
  return rounded * magnitude;
}

function ticks(minimum, maximum, count) {
  const span = maximum - minimum;
  const rawStep = span / Math.max(1, count);
  const magnitude = 10 ** Math.floor(Math.log10(rawStep || 1));
  const normalized = rawStep / magnitude;
  const step = (normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10) * magnitude;
  const values = [];
  for (let value = Math.ceil(minimum / step) * step; value <= maximum + step * .01; value += step) values.push(value);
  return values;
}

function currentRows() {
  return RAW_ROWS
    .filter(row => state.source === 'all' || row.source === state.source)
    .filter(row => Number.isFinite(row.pass_rate) && Number.isFinite(row[state.metric]))
    .map(row => ({ ...row, cost: row[state.metric], key: [row.model, row.effort || 'default', row.source].join('|') }));
}

function linePath(rows, x, y) {
  return rows.map((row, index) => (index ? 'L' : 'M') + x(row.cost).toFixed(2) + ',' + y(row.pass_rate).toFixed(2)).join(' ');
}

function setGeometry(rows) {
  const width = Math.max(300, Math.round(shell.clientWidth));
  const height = width < 560 ? 520 : 560;
  const margin = { top: 26, right: width < 560 ? 12 : 24, bottom: 66, left: width < 560 ? 54 : 66 };
  const plotWidth = Math.max(200, width - margin.left - margin.right);
  const plotHeight = height - margin.top - margin.bottom;
  const costs = rows.map(row => row.cost);
  const focusMax = quantile(costs, .95);
  const visibleMax = state.range === 'focus' ? focusMax : Math.max(...costs);
  const xMax = niceMax(visibleMax * 1.04);
  const rates = rows.filter(row => row.cost <= xMax).map(row => row.pass_rate);
  const rateMin = Math.max(0, Math.min(...rates) - .04);
  const rateMax = Math.min(1, Math.max(...rates) + .045);
  const yMin = Math.floor(rateMin * 20) / 20;
  const yMax = Math.max(yMin + .1, Math.ceil(rateMax * 20) / 20);
  const x = value => margin.left + value / xMax * plotWidth;
  const y = value => margin.top + (yMax - value) / (yMax - yMin) * plotHeight;
  svg.setAttribute('viewBox', '0 0 ' + width + ' ' + height);
  svg.style.height = height + 'px';
  document.getElementById('clip-rect').setAttribute('x', margin.left);
  document.getElementById('clip-rect').setAttribute('y', margin.top);
  document.getElementById('clip-rect').setAttribute('width', plotWidth);
  document.getElementById('clip-rect').setAttribute('height', plotHeight);
  geometry = { width, height, margin, plotWidth, plotHeight, xMax, yMin, yMax, x, y };
  return geometry;
}

function renderAxes(g) {
  const grid = document.getElementById('grid');
  const axes = document.getElementById('axes');
  grid.replaceChildren();
  axes.replaceChildren();
  const bottom = g.margin.top + g.plotHeight;
  const right = g.margin.left + g.plotWidth;
  ticks(0, g.xMax, 6).forEach(value => {
    const x = g.x(value);
    grid.append(svgNode('line', { class: 'grid-line', x1: x, x2: x, y1: g.margin.top, y2: bottom }));
    axes.append(svgNode('text', { class: 'axis-text', x, y: bottom + 21, 'text-anchor': 'middle' }, '$' + (value < 1 ? value.toFixed(1) : value.toFixed(0))));
  });
  ticks(g.yMin, g.yMax, 6).forEach(value => {
    const y = g.y(value);
    grid.append(svgNode('line', { class: 'grid-line', x1: g.margin.left, x2: right, y1: y, y2: y }));
    axes.append(svgNode('text', { class: 'axis-text', x: g.margin.left - 9, y: y + 4, 'text-anchor': 'end' }, Math.round(value * 100) + '%'));
  });
  axes.append(svgNode('line', { class: 'axis-line', x1: g.margin.left, x2: right, y1: bottom, y2: bottom }));
  axes.append(svgNode('line', { class: 'axis-line', x1: g.margin.left, x2: g.margin.left, y1: g.margin.top, y2: bottom }));
  axes.append(svgNode('text', { class: 'axis-title', x: g.margin.left + g.plotWidth / 2, y: g.height - 13, 'text-anchor': 'middle' }, state.metric === 'mean_cost' ? '平均成本（USD / task）' : '中位成本（USD / task）'));
  const yTitle = svgNode('text', { class: 'axis-title', x: -(g.margin.top + g.plotHeight / 2), y: 14, transform: 'rotate(-90)', 'text-anchor': 'middle' }, 'pass@1');
  axes.append(yTitle);
}

function renderTrajectories(rows, g) {
  const container = document.getElementById('trajectories');
  container.replaceChildren();
  const groups = new Map();
  rows.forEach(row => {
    if (!groups.has(row.model)) groups.set(row.model, []);
    groups.get(row.model).push(row);
  });
  groups.forEach((points, model) => {
    points.sort((a, b) => (effortRank.get(a.effort) ?? 99) - (effortRank.get(b.effort) ?? 99));
    if (points.length < 2) return;
    container.append(svgNode('path', { class: 'trajectory', d: linePath(points, g.x, g.y), stroke: colors.get(model) }));
  });
}

function tooltipHtml(row) {
  const effort = row.effort || 'default';
  const costLabel = state.metric === 'mean_cost' ? '平均成本' : '中位成本';
  return '<strong>' + escapeHtml(row.model) + ' · ' + escapeHtml(effort) + '</strong>'
    + '<div class="tooltip-line">pass@1 ' + (row.pass_rate * 100).toFixed(1) + '%</div>'
    + '<div class="tooltip-line">' + costLabel + ' $' + row.cost.toFixed(3) + '</div>'
    + '<div class="tooltip-line">' + escapeHtml(row.source) + '</div>';
}

function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character]);
}

function showTooltip(row, pointX, pointY) {
  tooltip.innerHTML = tooltipHtml(row);
  tooltip.classList.add('visible');
  const shellRect = shell.getBoundingClientRect();
  const svgRect = svg.getBoundingClientRect();
  const scaleX = svgRect.width / geometry.width;
  const scaleY = svgRect.height / geometry.height;
  const anchorX = svgRect.left - shellRect.left + pointX * scaleX;
  const anchorY = svgRect.top - shellRect.top + pointY * scaleY;
  const tooltipRect = tooltip.getBoundingClientRect();
  const left = Math.min(shell.clientWidth - tooltipRect.width - 6, Math.max(6, anchorX + 12));
  const top = Math.min(shell.clientHeight - tooltipRect.height - 6, Math.max(6, anchorY - tooltipRect.height - 10));
  tooltip.style.left = left + 'px';
  tooltip.style.top = top + 'px';
}

function renderPoints(rows, frontierKeys, g) {
  const container = document.getElementById('points');
  container.replaceChildren();
  rows.forEach(row => {
    const isPareto = frontierKeys.has(row.key);
    const circle = svgNode('circle', {
      class: 'point' + (isPareto ? ' is-pareto' : ''), cx: g.x(row.cost), cy: g.y(row.pass_rate),
      r: isPareto ? 5.5 : 4.5, fill: colors.get(row.model), 'aria-label': row.model + ' ' + (row.effort || 'default'),
    });
    circle.append(svgNode('title', {}, row.model + ' · ' + (row.effort || 'default') + ' · ' + (row.pass_rate * 100).toFixed(1) + '% · $' + row.cost.toFixed(3)));
    circle.addEventListener('mouseenter', () => showTooltip(row, g.x(row.cost), g.y(row.pass_rate)));
    circle.addEventListener('mouseleave', () => tooltip.classList.remove('visible'));
    container.append(circle);
  });
}

function renderLabels(frontier, g) {
  const container = document.getElementById('labels');
  container.replaceChildren();
  const bestByModel = new Map();
  frontier.forEach(row => {
    const current = bestByModel.get(row.model);
    if (!current || row.pass_rate > current.pass_rate) bestByModel.set(row.model, row);
  });
  const labels = [...bestByModel.values()].map(row => ({ row, pointX: g.x(row.cost), pointY: g.y(row.pass_rate) }))
    .sort((a, b) => a.pointY - b.pointY);
  const minGap = 17;
  labels.forEach((label, index) => {
    label.labelY = Math.max(g.margin.top + 11, label.pointY - 7);
    if (index > 0) label.labelY = Math.max(label.labelY, labels[index - 1].labelY + minGap);
  });
  if (labels.length) {
    const overflow = labels[labels.length - 1].labelY - (g.margin.top + g.plotHeight - 3);
    if (overflow > 0) labels.forEach(label => { label.labelY -= overflow; });
  }
  for (let index = labels.length - 2; index >= 0; index--) {
    labels[index].labelY = Math.min(labels[index].labelY, labels[index + 1].labelY - minGap);
  }
  labels.forEach(label => {
    const labelText = label.row.model + (label.row.effort ? ' · ' + label.row.effort : '');
    const textWidth = Math.min(170, labelText.length * 6.2);
    const placeLeft = label.pointX > g.margin.left + g.plotWidth * .68;
    const textX = placeLeft
      ? Math.max(g.margin.left + 2, label.pointX - textWidth - 10)
      : Math.min(g.margin.left + g.plotWidth - textWidth - 2, label.pointX + 10);
    const linkX = placeLeft ? textX + textWidth + 3 : textX - 3;
    container.append(svgNode('line', { class: 'label-link', x1: label.pointX, y1: label.pointY, x2: linkX, y2: label.labelY - 4 }));
    container.append(svgNode('text', { class: 'label', x: textX, y: label.labelY }, labelText));
  });
}

function renderLegend(rows) {
  const legend = document.getElementById('legend');
  legend.replaceChildren();
  [...new Set(rows.map(row => row.model))].sort().forEach(model => {
    const item = document.createElement('span');
    item.className = 'legend-item';
    const swatch = document.createElement('span');
    swatch.className = 'legend-line';
    swatch.style.background = colors.get(model);
    const label = document.createElement('span');
    label.textContent = model;
    item.append(swatch, label);
    legend.append(item);
  });
}

function renderTable(rows, frontierKeys) {
  const body = document.getElementById('table-body');
  body.replaceChildren();
  [...rows].sort((a, b) => b.pass_rate - a.pass_rate || a.cost - b.cost).forEach(row => {
    const cells = [row.model, row.effort || 'default', (row.pass_rate * 100).toFixed(1) + '%', '$' + row.cost.toFixed(3), frontierKeys.has(row.key) ? 'Pareto' : '', row.source];
    const tr = document.createElement('tr');
    cells.forEach((value, index) => {
      const td = document.createElement('td');
      td.textContent = value;
      if (index === 2 || index === 3) td.className = 'numeric';
      if (index === 4) td.className = 'col-pareto pareto-mark';
      if (index === 5) td.className = 'col-source';
      tr.append(td);
    });
    body.append(tr);
  });
  document.getElementById('table-count').textContent = rows.length + ' 个点';
  document.getElementById('cost-heading').textContent = state.metric === 'mean_cost' ? '平均成本' : '中位成本';
}

function update() {
  const allRows = currentRows();
  if (!allRows.length) return;
  const g = setGeometry(allRows);
  const visibleRows = allRows.filter(row => row.cost <= g.xMax + 1e-9);
  const frontier = paretoFrontier(allRows);
  const visibleFrontier = frontier.filter(row => row.cost <= g.xMax + 1e-9);
  const frontierKeys = new Set(frontier.map(row => row.key));
  renderAxes(g);
  renderTrajectories(visibleRows, g);
  document.getElementById('frontier').setAttribute('d', linePath(visibleFrontier, g.x, g.y));
  renderPoints(visibleRows, frontierKeys, g);
  renderLabels(visibleFrontier, g);
  renderLegend(visibleRows);
  renderTable(visibleRows, frontierKeys);
  const hidden = allRows.length - visibleRows.length;
  const modelCount = new Set(visibleRows.map(row => row.model)).size;
  document.getElementById('summary').textContent = visibleRows.length + ' 个点 · ' + modelCount + ' 个模型 · ' + visibleFrontier.length + ' 个 Pareto 点' + (hidden ? ' · 隐藏 ' + hidden + ' 个高成本点' : '');
  tooltip.classList.remove('visible');
}

function bindSegmented(id, field) {
  document.getElementById(id).addEventListener('click', event => {
    const button = event.target.closest('button[data-value]');
    if (!button) return;
    state[field] = button.dataset.value;
    document.querySelectorAll('#' + id + ' button').forEach(item => item.setAttribute('aria-pressed', String(item === button)));
    update();
  });
}

const sourceControl = document.getElementById('source-control');
const sources = [...new Set(RAW_ROWS.map(row => row.source))].sort();
[['all', '全部数据源'], ...sources.map(source => [source, source])].forEach(([value, label]) => {
  const option = document.createElement('option');
  option.value = value;
  option.textContent = label;
  sourceControl.append(option);
});
sourceControl.addEventListener('change', () => { state.source = sourceControl.value; update(); });
bindSegmented('metric-control', 'metric');
bindSegmented('range-control', 'range');
new ResizeObserver(() => update()).observe(shell);
update();
</script>
</body>
</html>
`
}

async function main() {
  const args = process.argv.slice(2)
  const input = resolve(optionValue(args, '--input', DEFAULT_INPUT))
  const output = resolve(optionValue(args, '--output', DEFAULT_OUTPUT))
  const raw = JSON.parse(await readFile(input, 'utf8'))
  const rows = validateRows(raw)
  await writeFile(output, renderBenchmarkChart(rows), 'utf8')
  console.log(`已生成 ${output}（${rows.length} 个数据点）`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(error => {
    console.error(`生成能力-成本曲线失败: ${error.message}`)
    process.exitCode = 1
  })
}

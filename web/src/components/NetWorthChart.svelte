<script lang="ts">
  import type { NetWorthMonth } from '../api/client';
  import { t, type MessageKey } from '../lib/i18n/index.svelte';
  import { linear, nearestIndex, niceTicks } from '../lib/chart';
  import { formatCents, formatWholeEuros } from '../lib/money';
  import { formatMonth } from '../lib/month';

  // Months oldest first.
  let { months }: { months: NetWorthMonth[] } = $props();

  // Geometry: the SVG scales to its container, so these are viewBox units.
  const width = 960;
  const height = 320;
  const margin = { top: 16, right: 44, bottom: 34, left: 84 };
  const plot = {
    left: margin.left,
    right: width - margin.right,
    top: margin.top,
    bottom: height - margin.bottom,
  };

  const series = [
    {
      id: 'assets',
      name: 'invest.networth.assets' as MessageKey,
      color: 'var(--series-1)',
      value: (m: NetWorthMonth) => m.assetsCents,
    },
    {
      id: 'owed',
      name: 'invest.networth.owed' as MessageKey,
      color: 'var(--series-2)',
      value: (m: NetWorthMonth) => m.creditCents,
    },
    {
      id: 'net',
      name: 'invest.networth.title' as MessageKey,
      color: 'var(--ink)',
      value: (m: NetWorthMonth) => m.netCents,
    },
  ];

  const ticks = $derived(
    niceTicks(
      Math.min(...months.flatMap((m) => series.map((s) => s.value(m))), 0),
      Math.max(...months.flatMap((m) => series.map((s) => s.value(m))), 0),
    ),
  );
  const y = $derived(linear([ticks[0], ticks[ticks.length - 1]], [plot.bottom, plot.top]));
  const x = $derived((i: number) =>
    months.length <= 1
      ? (plot.left + plot.right) / 2
      : plot.left + (i / (months.length - 1)) * (plot.right - plot.left),
  );
  const labelEvery = $derived(Math.max(1, Math.ceil(months.length / 8)));
  const path = (value: (m: NetWorthMonth) => number) =>
    months
      .map((m, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(value(m)).toFixed(1)}`)
      .join(' ');

  let hover = $state<number | null>(null);
  let svg: SVGSVGElement | undefined = $state();

  function point(event: PointerEvent) {
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    const px = ((event.clientX - box.left) / box.width) * width;
    hover = nearestIndex(px, plot.left, plot.right, months.length);
  }

  function key(event: KeyboardEvent) {
    const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (step === 0) return;
    event.preventDefault();
    hover = Math.min(months.length - 1, Math.max(0, (hover ?? months.length - 1) + step));
  }

  const hovered = $derived(hover === null ? undefined : months[hover]);
  const summary = $derived(
    t('invest.networth.chartSummary', {
      from: formatMonth(months[0]?.month ?? ''),
      to: formatMonth(months[months.length - 1]?.month ?? ''),
    }),
  );
</script>

<figure class="chart">
  <ul class="legend">
    {#each series as s}
      <li><span class="swatch" style:background={s.color}></span>{t(s.name)}</li>
    {/each}
  </ul>

  <div class="frame">
    <!-- The pointer readout is a convenience: the same figures are in the table below, and the arrow keys move the readout. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
    <svg
      bind:this={svg}
      viewBox="0 0 {width} {height}"
      role="img"
      aria-label={summary}
      tabindex="0"
      onpointermove={point}
      onpointerleave={() => (hover = null)}
      onkeydown={key}
      onblur={() => (hover = null)}
    >
      {#each ticks as tick}
        <line
          class={tick === 0 ? 'zero' : 'grid'}
          x1={plot.left}
          x2={plot.right}
          y1={y(tick)}
          y2={y(tick)}
        />
        <text class="axis" x={plot.left - 10} y={y(tick)} text-anchor="end" dy="0.32em">
          {formatWholeEuros(tick)}
        </text>
      {/each}

      {#each months as m, i}
        {#if i % labelEvery === 0}
          <text class="axis" x={x(i)} y={plot.bottom + 22} text-anchor="middle">
            {formatMonth(m.month)}
          </text>
        {/if}
      {/each}

      {#each series as s}
        <path class="line" class:net={s.id === 'net'} d={path(s.value)} stroke={s.color} />
        {#if months.length === 1}
          <circle class="dot" cx={x(0)} cy={y(s.value(months[0]))} r="4.5" fill={s.color} />
        {/if}
      {/each}

      {#if hover !== null && hovered}
        <line class="cursor" x1={x(hover)} x2={x(hover)} y1={plot.top} y2={plot.bottom} />
        {#each series as s}
          <circle class="dot" cx={x(hover)} cy={y(s.value(hovered))} r="4.5" fill={s.color} />
        {/each}
      {/if}
    </svg>

    {#if hovered && hover !== null}
      <div
        class="tip"
        style:left="{(x(hover) / width) * 100}%"
        style:transform="translateX({hover > months.length / 2 ? '-105%' : '5%'})"
        aria-hidden="true"
      >
        <strong>{formatMonth(hovered.month)}</strong>
        <dl>
          {#each series as s}
            <dt><span class="swatch" style:background={s.color}></span>{t(s.name)}</dt>
            <dd>{formatCents(s.value(hovered))}</dd>
          {/each}
        </dl>
      </div>
    {/if}
  </div>
</figure>

<style>
  .chart {
    margin: 0;
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 1.6rem;
    margin: 0 0 0.6rem;
    padding: 0;
    list-style: none;
    font-size: 0.9rem;
  }
  .swatch {
    display: inline-block;
    width: 1.3rem;
    height: 0.2rem;
    margin-right: 0.45rem;
    vertical-align: middle;
    border-radius: 0.1rem;
  }
  .frame {
    position: relative;
  }
  svg {
    display: block;
    width: 100%;
    height: auto;
    touch-action: pan-y;
  }
  svg:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 4px;
  }
  .grid {
    stroke: var(--rule);
    stroke-width: 1;
  }
  .zero {
    stroke: var(--ink);
    stroke-width: 1;
  }
  .axis {
    fill: var(--muted);
    font-size: 0.78rem;
  }
  .line {
    fill: none;
    stroke-width: 2.25;
    stroke-linejoin: round;
    stroke-linecap: round;
  }
  .line.net {
    stroke-width: 3.25;
  }
  .cursor {
    stroke: var(--muted);
    stroke-width: 1;
  }
  .dot {
    stroke: var(--bg);
    stroke-width: 2;
  }
  .tip {
    position: absolute;
    top: 0.5rem;
    min-width: 11rem;
    padding: 0.55rem 0.8rem;
    background: var(--bg);
    border: 1px solid var(--rule);
    border-radius: 4px;
    box-shadow: 0 2px 10px rgb(0 0 0 / 0.12);
    font-size: 0.9rem;
    pointer-events: none;
  }
  .tip dl {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.15rem 1rem;
    margin: 0.3rem 0 0;
  }
  .tip dt {
    color: var(--muted);
  }
  .tip dd {
    margin: 0;
    text-align: right;
  }
</style>

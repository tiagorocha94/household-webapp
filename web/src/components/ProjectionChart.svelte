<script lang="ts">
  import type { Projection } from '../api/client';
  import { linear, nearestIndex, niceTicks } from '../lib/chart';
  import { t } from '../lib/i18n/index.svelte';
  import { formatCents, formatWholeEuros } from '../lib/money';
  import { formatMonth } from '../lib/month';

  let { projection }: { projection: Projection } = $props();

  // Geometry: the SVG scales to its container, so these are viewBox units.
  const width = 960;
  const height = 340;
  const margin = { top: 16, right: 110, bottom: 34, left: 76 };
  const plot = {
    left: margin.left,
    right: width - margin.right,
    top: margin.top,
    bottom: height - margin.bottom,
  };

  const months = $derived(projection.months);
  const lastActual = $derived(months.reduce((last, m, i) => (m.isProjection ? last : i), -1));

  const series = $derived([
    {
      key: 'balanceCents' as const,
      name: t('goals.balance'),
      color: 'var(--series-1)',
      values: months.map((m) => m.balanceCents),
    },
    {
      key: 'unallocatedCents' as const,
      name: t('goals.free'),
      color: 'var(--series-2)',
      values: months.map((m) => m.unallocatedCents),
    },
  ]);

  const ticks = $derived(
    niceTicks(
      Math.min(...series.flatMap((s) => s.values)),
      Math.max(...series.flatMap((s) => s.values)),
    ),
  );
  const y = $derived(linear([ticks[0], ticks[ticks.length - 1]], [plot.bottom, plot.top]));
  const x = $derived((i: number) =>
    months.length <= 1
      ? plot.left
      : plot.left + (i / (months.length - 1)) * (plot.right - plot.left),
  );
  // About one label every three months, always including the first.
  const labelEvery = $derived(Math.max(1, Math.ceil(months.length / 8)));

  function path(values: number[], from: number, to: number): string {
    return values
      .slice(from, to + 1)
      .map((v, i) => `${i === 0 ? 'M' : 'L'}${x(from + i).toFixed(1)},${y(v).toFixed(1)}`)
      .join(' ');
  }

  // Hovering or keyboard focus picks one month; null hides the readout.
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
    hover = Math.min(months.length - 1, Math.max(0, (hover ?? lastActual) + step));
  }

  const hovered = $derived(hover === null ? undefined : months[hover]);
  const summary = $derived(
    t('goals.chart.projectionSummary', {
      from: formatMonth(months[0]?.month ?? ''),
      to: formatMonth(months[months.length - 1]?.month ?? ''),
    }),
  );
</script>

{#if months.length === 0}
  <p class="muted">{t('goals.chart.nothingToProject')}</p>
{:else}
  <figure class="chart">
    <ul class="legend">
      {#each series as s}
        <li><span class="swatch" style:background={s.color}></span>{s.name}</li>
      {/each}
      <li class="muted">
        <span class="swatch dashed"></span>{t('goals.chart.projectedLegend', {
          amount: formatCents(projection.monthlyDepositCents),
        })}
      </li>
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
        {#if lastActual < months.length - 1}
          <rect
            class="future"
            x={x(Math.max(lastActual, 0))}
            y={plot.top}
            width={plot.right - x(Math.max(lastActual, 0))}
            height={plot.bottom - plot.top}
          />
        {/if}

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
          {#if lastActual >= 0}
            <path class="line" d={path(s.values, 0, lastActual)} stroke={s.color} />
          {/if}
          {#if lastActual < months.length - 1}
            <path
              class="line projected"
              d={path(s.values, Math.max(lastActual, 0), months.length - 1)}
              stroke={s.color}
            />
          {/if}
          <text class="end" x={plot.right + 8} y={y(s.values[s.values.length - 1])} dy="0.32em">
            {s.name}
          </text>
        {/each}

        {#if hover !== null}
          <line class="cursor" x1={x(hover)} x2={x(hover)} y1={plot.top} y2={plot.bottom} />
          {#each series as s}
            <circle class="dot" cx={x(hover)} cy={y(s.values[hover])} r="4.5" fill={s.color} />
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
          {#if hovered.isProjection}<span class="muted"> {t('goals.chart.projected')}</span>{/if}
          <dl>
            {#each series as s}
              <dt><span class="swatch" style:background={s.color}></span>{s.name}</dt>
              <dd>{formatCents(hovered[s.key])}</dd>
            {/each}
            <dt>{t('goals.chart.deposits')}</dt>
            <dd>{formatCents(hovered.depositsCents)}</dd>
            <dt>{t('goals.chart.allocatedToGoals')}</dt>
            <dd>{formatCents(hovered.allocationsCents)}</dd>
          </dl>
        </div>
      {/if}
    </div>

    <details class="table-view">
      <summary>{t('goals.chart.showTable')}</summary>
      <table class="ledger">
        <thead>
          <tr>
            <th>{t('goals.month')}</th>
            <th class="num">{t('goals.chart.deposits')}</th>
            <th class="num">{t('goals.allocated')}</th>
            <th class="num">{t('goals.spent')}</th>
            <th class="num">{t('goals.chart.released')}</th>
            <th class="num">{t('goals.chart.covered')}</th>
            <th class="num">{t('goals.balance')}</th>
            <th class="num">{t('goals.free')}</th>
          </tr>
        </thead>
        <tbody>
          {#each months as m (m.month)}
            <tr class:projected-row={m.isProjection}>
              <td
                >{formatMonth(m.month)}{m.isProjection
                  ? ` (${t('goals.chart.projected')})`
                  : ''}</td
              >
              <td class="num">{formatCents(m.depositsCents)}</td>
              <td class="num">{formatCents(m.allocationsCents)}</td>
              <td class="num">{formatCents(m.expensesCents)}</td>
              <td class="num">{formatCents(m.releasedCents)}</td>
              <td class="num">{formatCents(m.coveredCents)}</td>
              <td class="num">{formatCents(m.balanceCents)}</td>
              <td class="num {m.unallocatedCents < 0 ? 'out' : ''}">
                {formatCents(m.unallocatedCents)}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </details>
  </figure>
{/if}

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
  .swatch.dashed {
    height: 0;
    background: none;
    border-top: 2px dashed var(--muted);
    border-radius: 0;
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
  .future {
    fill: var(--rule);
    opacity: 0.35;
  }
  .axis {
    fill: var(--muted);
    font-size: 0.78rem;
  }
  .end {
    fill: var(--ink);
    font-size: 0.8125rem;
  }
  .line {
    fill: none;
    stroke-width: 2.25;
    stroke-linejoin: round;
    stroke-linecap: round;
  }
  .line.projected {
    stroke-dasharray: 6 5;
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
    min-width: 12rem;
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
  .table-view {
    margin-top: 1rem;
  }
  .table-view summary {
    cursor: pointer;
    color: var(--accent);
    margin-bottom: 0.6rem;
  }
  .projected-row td {
    color: var(--muted);
  }
</style>

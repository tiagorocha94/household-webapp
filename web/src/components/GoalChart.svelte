<script lang="ts">
  import { linear, niceTicks } from '../lib/chart';
  import type { GoalSeries } from '../lib/goals';
  import { t } from '../lib/i18n/index.svelte';
  import { formatCents, formatWholeEuros } from '../lib/money';
  import { formatMonth } from '../lib/month';

  let { series }: { series: GoalSeries } = $props();

  // Geometry: the SVG scales to its container, so these are viewBox units.
  const width = 960;
  const height = 300;
  const margin = { top: 24, right: 64, bottom: 34, left: 76 };
  const plot = {
    left: margin.left,
    right: width - margin.right,
    top: margin.top,
    bottom: height - margin.bottom,
  };

  const count = $derived(series.months.length);
  const all = $derived([...series.balance, ...series.planned, 0]);
  const ticks = $derived(niceTicks(Math.min(...all), Math.max(...all)));
  const y = $derived(linear([ticks[0], ticks[ticks.length - 1]], [plot.bottom, plot.top]));
  const x = $derived((i: number) =>
    count <= 1 ? plot.left : plot.left + (i / (count - 1)) * (plot.right - plot.left),
  );
  const labelEvery = $derived(Math.max(1, Math.ceil(count / 8)));

  function path(values: number[]): string {
    return values
      .map((v, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(v).toFixed(1)}`)
      .join(' ');
  }

  const lastIndex = $derived(series.balance.length - 1);
  const lastValue = $derived(series.balance[lastIndex] ?? 0);
  const summary = $derived(
    t('goals.chart.goalSummary', {
      from: formatMonth(series.months[0] ?? ''),
      to: formatMonth(series.months[lastIndex] ?? ''),
      amount: formatCents(lastValue),
    }),
  );
</script>

{#if count > 0 && series.balance.length > 0}
  <figure class="chart">
    <ul class="legend">
      <li><span class="swatch"></span>{t('goals.balance')}</li>
      <li class="muted"><span class="swatch dotted"></span>{t('goals.chart.plannedIfNone')}</li>
    </ul>
    <svg viewBox="0 0 {width} {height}" role="img" aria-label={summary}>
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
      {#each series.months as m, i}
        {#if i % labelEvery === 0}
          <text class="axis" x={x(i)} y={plot.bottom + 22} text-anchor="middle">
            {formatMonth(m)}
          </text>
        {/if}
      {/each}

      <path class="planned" d={path(series.planned)} />
      <path class="balance" d={path(series.balance)} />
      {#each series.balance as v, i}
        {#if v < 0}<circle class="dot negative" cx={x(i)} cy={y(v)} r="4.5" />{/if}
      {/each}
      <circle class="dot" cx={x(lastIndex)} cy={y(lastValue)} r="5.5" />
      <text
        class="end"
        x={x(lastIndex)}
        y={y(lastValue) - 12}
        text-anchor={lastIndex > count / 2 ? 'end' : 'start'}
      >
        {formatCents(lastValue)}
      </text>
    </svg>

    <details class="table-view">
      <summary>{t('goals.chart.showTable')}</summary>
      <table class="ledger">
        <thead>
          <tr>
            <th>{t('goals.month')}</th>
            <th class="num">{t('goals.balance')}</th>
            <th class="num">{t('goals.chart.planned')}</th>
          </tr>
        </thead>
        <tbody>
          {#each series.months as m, i}
            <tr>
              <td>{formatMonth(m)}</td>
              <td class="num {(series.balance[i] ?? 0) < 0 ? 'out' : ''}">
                {i < series.balance.length ? formatCents(series.balance[i]) : ''}
              </td>
              <td class="num muted">{formatCents(series.planned[i])}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </details>
  </figure>
{/if}

<style>
  .chart {
    margin: 1.2rem 0 0;
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
    background: var(--series-1);
  }
  .swatch.dotted {
    height: 0;
    background: none;
    border-top: 2px dotted var(--muted);
    border-radius: 0;
  }
  svg {
    display: block;
    width: 100%;
    height: auto;
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
  .end {
    fill: var(--ink);
    font-size: 0.8125rem;
    font-weight: 600;
  }
  .balance,
  .planned {
    fill: none;
    stroke-linejoin: round;
    stroke-linecap: round;
  }
  .balance {
    stroke: var(--series-1);
    stroke-width: 2.25;
  }
  .planned {
    stroke: var(--muted);
    stroke-width: 2;
    stroke-dasharray: 1 6;
  }
  .dot {
    fill: var(--series-1);
    stroke: var(--bg);
    stroke-width: 2;
  }
  .dot.negative {
    fill: var(--out);
  }
  .table-view {
    margin-top: 1rem;
  }
  .table-view summary {
    cursor: pointer;
    color: var(--accent);
    margin-bottom: 0.6rem;
  }
</style>

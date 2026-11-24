<script lang="ts">
  import type { CashflowMonth } from '../api/client';
  import { formatChange } from '../lib/cashflow';
  import { linear, niceTicks } from '../lib/chart';
  import { t, type MessageKey } from '../lib/i18n/index.svelte';
  import { formatCents, formatWholeEuros } from '../lib/money';
  import { formatMonth } from '../lib/month';

  // Months oldest first.
  let { months }: { months: CashflowMonth[] } = $props();

  // Geometry: the SVG scales to its container, so these are viewBox units.
  const width = 960;
  const height = 320;
  const margin = { top: 16, right: 24, bottom: 34, left: 76 };
  const plot = {
    left: margin.left,
    right: width - margin.right,
    top: margin.top,
    bottom: height - margin.bottom,
  };

  const income = { name: 'cashflow.income' as MessageKey, color: 'var(--series-1)' };
  const expenses = { name: 'cashflow.expenses' as MessageKey, color: 'var(--series-2)' };

  const ticks = $derived(
    niceTicks(0, Math.max(...months.flatMap((m) => [m.incomeCents, m.expenseCents]), 0)),
  );
  const y = $derived(linear([ticks[0], ticks[ticks.length - 1]], [plot.bottom, plot.top]));
  const group = $derived((plot.right - plot.left) / Math.max(months.length, 1));
  const barWidth = $derived(Math.min(30, group * 0.36));
  const center = $derived((i: number) => plot.left + (i + 0.5) * group);
  const labelEvery = $derived(Math.max(1, Math.ceil(months.length / 8)));

  let hover = $state<number | null>(null);
  let svg: SVGSVGElement | undefined = $state();

  function point(event: PointerEvent) {
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    const px = ((event.clientX - box.left) / box.width) * width;
    const index = Math.floor((px - plot.left) / group);
    hover = index < 0 || index >= months.length ? null : index;
  }

  function key(event: KeyboardEvent) {
    const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (step === 0) return;
    event.preventDefault();
    hover = Math.min(months.length - 1, Math.max(0, (hover ?? months.length - 1) + step));
  }

  const hovered = $derived(hover === null ? undefined : months[hover]);
  const summary = $derived(
    t('cashflow.chartSummary', {
      from: formatMonth(months[0]?.month ?? ''),
      to: formatMonth(months[months.length - 1]?.month ?? ''),
    }),
  );
</script>

{#if months.length === 0}
  <p class="muted">{t('cashflow.nothingToChart')}</p>
{:else}
  <figure class="chart">
    <ul class="legend">
      {#each [income, expenses] as s}
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

        {#if hover !== null}
          <rect
            class="cursor"
            x={plot.left + hover * group}
            y={plot.top}
            width={group}
            height={plot.bottom - plot.top}
          />
        {/if}

        {#each months as m, i (m.month)}
          {#if i % labelEvery === 0}
            <text class="axis" x={center(i)} y={plot.bottom + 22} text-anchor="middle">
              {formatMonth(m.month)}
            </text>
          {/if}
          <rect
            x={center(i) - barWidth - 1}
            y={y(m.incomeCents)}
            width={barWidth}
            height={y(0) - y(m.incomeCents)}
            fill={income.color}
          />
          <rect
            x={center(i) + 1}
            y={y(m.expenseCents)}
            width={barWidth}
            height={y(0) - y(m.expenseCents)}
            fill={expenses.color}
          />
        {/each}
      </svg>

      {#if hovered && hover !== null}
        <div
          class="tip"
          style:left="{(center(hover) / width) * 100}%"
          style:transform="translateX({hover > months.length / 2 ? '-105%' : '5%'})"
          aria-hidden="true"
        >
          <strong>{formatMonth(hovered.month)}</strong>
          <dl>
            <dt>
              <span class="swatch" style:background={income.color}></span>{t('cashflow.income')}
            </dt>
            <dd>{formatCents(hovered.incomeCents)}</dd>
            <dt>
              <span class="swatch" style:background={expenses.color}></span>{t('cashflow.expenses')}
            </dt>
            <dd>{formatCents(hovered.expenseCents)}</dd>
            <dt>{t('cashflow.leftOver')}</dt>
            <dd>{formatChange(hovered.netCents)}</dd>
          </dl>
        </div>
      {/if}
    </div>

    <details class="table-view">
      <summary>{t('cashflow.showTable')}</summary>
      <table class="ledger">
        <thead>
          <tr>
            <th>{t('cashflow.month')}</th>
            <th class="num">{t('cashflow.income')}</th>
            <th class="num">{t('cashflow.expenses')}</th>
            <th class="num">{t('cashflow.leftOver')}</th>
          </tr>
        </thead>
        <tbody>
          {#each months as m (m.month)}
            <tr>
              <td>{formatMonth(m.month)}</td>
              <td class="num">{formatCents(m.incomeCents)}</td>
              <td class="num">{formatCents(m.expenseCents)}</td>
              <td class="num">{formatCents(m.netCents)}</td>
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
    height: 0.6rem;
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
  .cursor {
    fill: var(--rule);
    opacity: 0.5;
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
  .table-view {
    margin-top: 1rem;
  }
  .table-view summary {
    cursor: pointer;
    color: var(--accent);
    margin-bottom: 0.6rem;
  }
</style>

<script lang="ts">
  import { linear, nearestIndex, niceTicks } from '../lib/chart';
  import type { HistoryPoint } from '../lib/investments';
  import { localeTag, t } from '../lib/i18n/index.svelte';
  import { formatCents, formatWholeEuros } from '../lib/money';
  import { formatMonth } from '../lib/month';

  let {
    history,
    valueLabel: valueLabelProp,
    inLabel: inLabelProp,
  }: { history: HistoryPoint[]; valueLabel?: string; inLabel?: string } = $props();
  const valueLabel = $derived(valueLabelProp ?? t('goals.chart.value'));
  const inLabel = $derived(inLabelProp ?? t('goals.chart.moneyIn'));

  // Geometry: the SVG scales to its container, so these are viewBox units.
  const width = 960;
  const height = 320;
  const margin = { top: 16, right: 90, bottom: 34, left: 76 };
  const plot = {
    left: margin.left,
    right: width - margin.right,
    top: margin.top,
    bottom: height - margin.bottom,
  };

  const series = $derived([
    {
      name: valueLabel,
      color: 'var(--series-1)',
      values: history.map((p) => p.valueCents),
    },
    {
      name: inLabel,
      color: 'var(--series-2)',
      values: history.map((p) => p.allocatedCents as number | null),
    },
  ]);

  const all = $derived(series.flatMap((s) => s.values).filter((v): v is number => v !== null));
  const ticks = $derived(niceTicks(Math.min(...all), Math.max(...all)));
  const y = $derived(linear([ticks[0], ticks[ticks.length - 1]], [plot.bottom, plot.top]));
  const x = $derived((i: number) =>
    history.length <= 1
      ? (plot.left + plot.right) / 2
      : plot.left + (i / (history.length - 1)) * (plot.right - plot.left),
  );
  // The value line is solid between observed snapshots and dashed over months
  // that are estimated on the straight line between two of them.
  const valuePaths = $derived.by(() => {
    let solid = '';
    let dashed = '';
    let solidEnd = -1;
    let dashedEnd = -1;
    for (let i = 0; i + 1 < history.length; i++) {
      const a = history[i];
      const b = history[i + 1];
      if (a.valueCents === null || b.valueCents === null) continue;
      const to = `L${x(i + 1).toFixed(1)},${y(b.valueCents).toFixed(1)}`;
      const from = `M${x(i).toFixed(1)},${y(a.valueCents).toFixed(1)} ${to}`;
      if (a.estimated || b.estimated) {
        dashed += dashedEnd === i ? ` ${to}` : ` ${from}`;
        dashedEnd = i + 1;
      } else {
        solid += solidEnd === i ? ` ${to}` : ` ${from}`;
        solidEnd = i + 1;
      }
    }
    return { solid: solid.trim(), dashed: dashed.trim() };
  });
  const hasEstimates = $derived(history.some((p) => p.estimated));

  const labelEvery = $derived(Math.max(1, Math.ceil(history.length / 8)));

  // A line skips months without a value, so it starts at the first snapshot.
  function path(values: (number | null)[]): string {
    return values
      .map((v, i) => (v === null ? null : { i, v }))
      .filter((p) => p !== null)
      .map((p, n) => `${n === 0 ? 'M' : 'L'}${x(p.i).toFixed(1)},${y(p.v).toFixed(1)}`)
      .join(' ');
  }

  let hover = $state<number | null>(null);
  let svg: SVGSVGElement | undefined = $state();

  function point(event: PointerEvent) {
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    const px = ((event.clientX - box.left) / box.width) * width;
    hover = nearestIndex(px, plot.left, plot.right, history.length);
  }

  function key(event: KeyboardEvent) {
    const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (step === 0) return;
    event.preventDefault();
    hover = Math.min(history.length - 1, Math.max(0, (hover ?? history.length - 1) + step));
  }

  const hovered = $derived(hover === null ? undefined : history[hover]);
  const summary = $derived(
    t('goals.chart.historySummary', {
      value: valueLabel,
      moneyIn: inLabel.toLocaleLowerCase(localeTag()),
      from: formatMonth(history[0]?.month ?? ''),
      to: formatMonth(history[history.length - 1]?.month ?? ''),
    }),
  );
</script>

{#if history.length === 0}
  <p class="muted">{t('goals.chart.noHistory')}</p>
{:else}
  <figure class="chart">
    <ul class="legend">
      {#each series as s}
        <li><span class="swatch" style:background={s.color}></span>{s.name}</li>
      {/each}
      <li class="muted"><span class="swatch marker-swatch"></span>{t('goals.chart.snapshot')}</li>
      {#if hasEstimates}
        <li class="muted">
          <span class="swatch dashed"></span>{t('goals.chart.estimatedLegend')}
        </li>
      {/if}
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

        {#each history as p, i}
          {#if i % labelEvery === 0}
            <text class="axis" x={x(i)} y={plot.bottom + 22} text-anchor="middle">
              {formatMonth(p.month)}
            </text>
          {/if}
        {/each}

        <path class="line" d={path(series[1].values)} stroke={series[1].color} />
        <path class="line" d={valuePaths.solid} stroke={series[0].color} />
        <path class="line estimated" d={valuePaths.dashed} stroke={series[0].color} />
        <!-- A marker sits on every month that has a real snapshot; estimated months have none. -->
        {#each history as p, i}
          {#if p.valueCents !== null && !p.estimated}
            <circle class="marker" cx={x(i)} cy={y(p.valueCents)} r="4" fill={series[0].color} />
          {/if}
        {/each}
        {#each series as s}
          {#if history.length === 1}
            {#each s.values as v}
              {#if v !== null}<circle class="dot" cx={x(0)} cy={y(v)} r="4.5" fill={s.color} />{/if}
            {/each}
          {/if}
        {/each}

        {#if hover !== null}
          <line class="cursor" x1={x(hover)} x2={x(hover)} y1={plot.top} y2={plot.bottom} />
          {#each series as s}
            {@const v = s.values[hover]}
            {#if v !== null}
              <circle class="dot" cx={x(hover)} cy={y(v)} r="4.5" fill={s.color} />
            {/if}
          {/each}
        {/if}
      </svg>

      {#if hovered && hover !== null}
        <div
          class="tip"
          style:left="{(x(hover) / width) * 100}%"
          style:transform="translateX({hover > history.length / 2 ? '-105%' : '5%'})"
          aria-hidden="true"
        >
          <strong>{formatMonth(hovered.month)}</strong>
          {#if hovered.estimated}<span class="muted"> {t('goals.chart.estimated')}</span>{/if}
          <dl>
            <dt><span class="swatch" style:background={series[0].color}></span>{valueLabel}</dt>
            <dd>{hovered.valueCents === null ? '—' : formatCents(hovered.valueCents)}</dd>
            <dt><span class="swatch" style:background={series[1].color}></span>{inLabel}</dt>
            <dd>{formatCents(hovered.allocatedCents)}</dd>
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
            <th class="num">{inLabel}</th>
            <th class="num">{valueLabel}</th>
          </tr>
        </thead>
        <tbody>
          {#each history as p (p.month)}
            <tr>
              <td>{formatMonth(p.month)}</td>
              <td class="num">{formatCents(p.allocatedCents)}</td>
              <td class="num">
                {p.valueCents === null ? '—' : formatCents(p.valueCents)}{p.estimated
                  ? ` (${t('goals.chart.estimatedShort')})`
                  : ''}
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
    border-top: 2px dashed var(--series-1);
    border-radius: 0;
  }
  .swatch.marker-swatch {
    width: 0.6rem;
    height: 0.6rem;
    margin-right: 0.55rem;
    background: var(--series-1);
    border-radius: 50%;
  }
  .marker {
    stroke: var(--bg);
    stroke-width: 2;
  }
  .line.estimated {
    stroke-dasharray: 6 5;
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
  .table-view {
    margin-top: 1rem;
  }
  .table-view summary {
    cursor: pointer;
    color: var(--accent);
    margin-bottom: 0.6rem;
  }
</style>

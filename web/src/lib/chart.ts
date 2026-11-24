// Scales for the SVG charts. Pure, so they are tested without a browser.

/**
 * Evenly spaced round values that cover min to max, such as 0, 250000, 500000.
 * The first and last tick sit within one step of the data, so the axis hugs it
 * instead of always reaching zero.
 */
export function niceTicks(min: number, max: number, target = 5): number[] {
  let lo = Math.min(min, max);
  let hi = Math.max(min, max);
  // A flat series gets a little room around it, so its line is not pinned to
  // the edge and the axis does not stretch down to zero.
  if (lo === hi) {
    const pad = Math.max(Math.abs(hi) * 0.05, 100);
    lo -= pad;
    hi += pad;
  }
  const range = hi - lo;
  const rough = range / Math.max(target - 1, 1);
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  // Money axes are labelled in whole euros, so a step is never below one euro.
  const step = Math.max(
    100,
    ([1, 2, 2.5, 5, 10].find((m) => m * magnitude >= rough) ?? 10) * magnitude,
  );
  const bottom = Math.floor(lo / step) * step;
  let top = Math.ceil(hi / step) * step;
  if (top === bottom) top += step; // a flat range still needs an axis
  const ticks: number[] = [];
  for (let v = bottom; v <= top + step / 2; v += step) ticks.push(Math.round(v));
  return ticks;
}

/** A linear scale from the domain to the range. */
export function linear(
  domain: readonly [number, number],
  range: readonly [number, number],
): (value: number) => number {
  const span = domain[1] - domain[0] || 1;
  return (value) => range[0] + ((value - domain[0]) / span) * (range[1] - range[0]);
}

/** The index of the point nearest to x, for points evenly spaced from left to right. */
export function nearestIndex(x: number, left: number, right: number, count: number): number {
  if (count <= 1 || right <= left) return 0;
  const t = (x - left) / (right - left);
  return Math.min(count - 1, Math.max(0, Math.round(t * (count - 1))));
}

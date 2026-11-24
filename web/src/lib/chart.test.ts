import { describe, expect, it } from 'vitest';
import { linear, nearestIndex, niceTicks } from './chart';

describe('niceTicks', () => {
  it('covers the range with round, evenly spaced values', () => {
    const ticks = niceTicks(0, 950000);
    expect(ticks[0]).toBeLessThanOrEqual(0);
    expect(ticks[ticks.length - 1]).toBeGreaterThanOrEqual(950000);
    const steps = new Set(ticks.slice(1).map((t, i) => t - ticks[i]));
    expect(steps.size).toBe(1);
  });
  it('includes zero when the range crosses it', () => {
    expect(niceTicks(-120000, 480000)).toContain(0);
  });
  it('copes with a flat range', () => {
    expect(niceTicks(0, 0).length).toBeGreaterThan(1);
  });
  it('hugs the data instead of reaching zero', () => {
    const ticks = niceTicks(98000, 123500);
    const step = ticks[1] - ticks[0];
    expect(ticks[0]).toBeGreaterThan(0);
    expect(98000 - ticks[0]).toBeLessThan(step);
    expect(ticks[ticks.length - 1] - 123500).toBeLessThan(step);
  });
  it('keeps a flat, non-zero series near its value', () => {
    const ticks = niceTicks(101500, 101500);
    expect(ticks[0]).toBeGreaterThan(90000);
    expect(ticks[ticks.length - 1]).toBeLessThan(115000);
  });
  it('shows a small change as a visible spread of ticks', () => {
    const ticks = niceTicks(100000, 100900);
    expect(ticks[ticks.length - 1] - ticks[0]).toBeLessThan(5000);
  });
});

describe('linear', () => {
  it('maps the domain onto the range, including an inverted one', () => {
    const y = linear([0, 100], [300, 0]);
    expect(y(0)).toBe(300);
    expect(y(100)).toBe(0);
    expect(y(50)).toBe(150);
  });
});

describe('nearestIndex', () => {
  it('picks the closest point and stays inside the data', () => {
    expect(nearestIndex(0, 0, 100, 5)).toBe(0);
    expect(nearestIndex(60, 0, 100, 5)).toBe(2);
    expect(nearestIndex(999, 0, 100, 5)).toBe(4);
    expect(nearestIndex(-5, 0, 100, 5)).toBe(0);
  });
});

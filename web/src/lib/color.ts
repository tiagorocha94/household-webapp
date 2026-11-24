// Picks a readable icon colour for a coloured badge.

const INK = '#15202b';
const WHITE = '#ffffff';

function channel(value: number): number {
  const v = value / 255;
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

/** The WCAG relative luminance of a `#RRGGBB` colour, 0 (black) to 1 (white). */
export function luminance(hex: string): number {
  if (!/^#[0-9a-fA-F]{6}$/.test(hex)) throw new Error(`invalid colour: ${hex}`);
  const r = channel(parseInt(hex.slice(1, 3), 16));
  const g = channel(parseInt(hex.slice(3, 5), 16));
  const b = channel(parseInt(hex.slice(5, 7), 16));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/** The WCAG contrast ratio between two colours, 1 to 21. */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

/**
 * White for a dark or medium background, dark ink for a light one. White is kept
 * while it still reaches 3:1, the minimum for graphics; below that, as on
 * yellow, an icon would nearly disappear.
 */
export function iconColorOn(background: string): string {
  return contrast(WHITE, background) >= 3 ? WHITE : INK;
}

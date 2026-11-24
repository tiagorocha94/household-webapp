// How the page looks, chosen by the person on the settings page and kept in this browser only
// (local storage), never on the server. Every choice starts at its default: the theme follows the
// system, the text is normal, the rows comfortable and the accent green. A default is stored as
// nothing at all, so "System" is the absence of a choice.
//
// The page reads the choice from attributes on the root element (see src/app.css). `public/early.js`
// does the same before the page paints, so there is no flash; a test keeps the two in step.

export type Theme = 'system' | 'light' | 'dark';
export type TextSize = 'normal' | 'large' | 'larger';
export type RowDensity = 'comfortable' | 'compact';
export type Accent = 'green' | 'blue' | 'purple' | 'pink';

export interface Appearance {
  theme: Theme;
  textSize: TextSize;
  rows: RowDensity;
  accent: Accent;
}

export const DEFAULT_APPEARANCE: Appearance = {
  theme: 'system',
  textSize: 'normal',
  rows: 'comfortable',
  accent: 'green',
};

export const THEMES: readonly Theme[] = ['system', 'light', 'dark'];
export const TEXT_SIZES: readonly TextSize[] = ['normal', 'large', 'larger'];
export const ROW_DENSITIES: readonly RowDensity[] = ['comfortable', 'compact'];
export const ACCENTS: readonly Accent[] = ['green', 'blue', 'purple', 'pink'];

/** The local storage keys, one per choice. */
export const APPEARANCE_KEYS = {
  theme: 'household.theme',
  textSize: 'household.textSize',
  rows: 'household.rows',
  accent: 'household.accent',
} as const;

function pick<T extends string>(choices: readonly T[], stored: string | null, fallback: T): T {
  return choices.find((choice) => choice === stored) ?? fallback;
}

/** Reads the choices out of what was stored; anything unknown or missing is the default. */
export function parseAppearance(read: (key: string) => string | null): Appearance {
  return {
    theme: pick(THEMES, read(APPEARANCE_KEYS.theme), DEFAULT_APPEARANCE.theme),
    textSize: pick(TEXT_SIZES, read(APPEARANCE_KEYS.textSize), DEFAULT_APPEARANCE.textSize),
    rows: pick(ROW_DENSITIES, read(APPEARANCE_KEYS.rows), DEFAULT_APPEARANCE.rows),
    accent: pick(ACCENTS, read(APPEARANCE_KEYS.accent), DEFAULT_APPEARANCE.accent),
  };
}

/** The attributes of the root element for the choices; null means the attribute is absent. */
export function rootAttributes(a: Appearance): Record<string, string | null> {
  return {
    'data-theme': a.theme === 'system' ? null : a.theme,
    'data-text-size': a.textSize === 'normal' ? null : a.textSize,
    'data-rows': a.rows === 'comfortable' ? null : a.rows,
    'data-accent': a.accent === 'green' ? null : a.accent,
  };
}

/** Puts the choices on the page. */
export function applyAppearance(
  a: Appearance,
  root: Pick<HTMLElement, 'setAttribute' | 'removeAttribute'>,
): void {
  for (const [name, value] of Object.entries(rootAttributes(a))) {
    if (value === null) root.removeAttribute(name);
    else root.setAttribute(name, value);
  }
}

/** True when a choice is its default, which is stored as nothing. */
export function isDefault<K extends keyof Appearance>(key: K, value: Appearance[K]): boolean {
  return DEFAULT_APPEARANCE[key] === value;
}

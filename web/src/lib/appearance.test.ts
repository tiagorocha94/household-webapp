import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import {
  ACCENTS,
  APPEARANCE_KEYS,
  DEFAULT_APPEARANCE,
  applyAppearance,
  isDefault,
  parseAppearance,
  rootAttributes,
  type Appearance,
} from './appearance';
import { ACCENT_SWATCHES } from './settings';

const none = () => null;

describe('parseAppearance', () => {
  it('starts at the defaults: the system theme, normal text, comfortable rows, green', () => {
    expect(parseAppearance(none)).toEqual(DEFAULT_APPEARANCE);
  });

  it('reads what was stored', () => {
    const stored: Record<string, string> = {
      [APPEARANCE_KEYS.theme]: 'dark',
      [APPEARANCE_KEYS.textSize]: 'larger',
      [APPEARANCE_KEYS.rows]: 'compact',
      [APPEARANCE_KEYS.accent]: 'purple',
    };
    expect(parseAppearance((key) => stored[key] ?? null)).toEqual({
      theme: 'dark',
      textSize: 'larger',
      rows: 'compact',
      accent: 'purple',
    });
  });

  it('ignores anything it does not know', () => {
    expect(parseAppearance(() => 'rainbow')).toEqual(DEFAULT_APPEARANCE);
  });
});

describe('rootAttributes', () => {
  it('leaves every attribute out at the defaults, so the system decides', () => {
    expect(rootAttributes(DEFAULT_APPEARANCE)).toEqual({
      'data-theme': null,
      'data-text-size': null,
      'data-rows': null,
      'data-accent': null,
    });
  });

  it('sets the attribute of each choice', () => {
    expect(
      rootAttributes({ theme: 'light', textSize: 'large', rows: 'compact', accent: 'pink' }),
    ).toEqual({
      'data-theme': 'light',
      'data-text-size': 'large',
      'data-rows': 'compact',
      'data-accent': 'pink',
    });
  });
});

describe('applyAppearance', () => {
  it('sets and removes the attributes', () => {
    const attributes = new Map<string, string>([['data-theme', 'dark']]);
    const root = {
      setAttribute: (name: string, value: string) => attributes.set(name, value),
      removeAttribute: (name: string) => attributes.delete(name),
    };
    applyAppearance({ ...DEFAULT_APPEARANCE, accent: 'blue' }, root);
    expect(Object.fromEntries(attributes)).toEqual({ 'data-accent': 'blue' });
  });
});

describe('isDefault', () => {
  it('is true only for the default of each choice', () => {
    expect(isDefault('theme', 'system')).toBe(true);
    expect(isDefault('theme', 'dark')).toBe(false);
    expect(isDefault('accent', 'green')).toBe(true);
    expect(isDefault('rows', 'compact')).toBe(false);
  });
});

describe('public/early.js', () => {
  // The script that runs before the page paints must give the same attributes as the code above.
  const script = readFileSync('public/early.js', 'utf8');

  function run(stored: Record<string, string>): Record<string, string> {
    const attributes: Record<string, string> = {};
    const fakeWindow = { localStorage: { getItem: (key: string) => stored[key] ?? null } };
    const fakeDocument = {
      documentElement: {
        setAttribute: (name: string, value: string) => (attributes[name] = value),
      },
    };
    new Function('window', 'document', script)(fakeWindow, fakeDocument);
    return attributes;
  }

  const cases: Appearance[] = [
    DEFAULT_APPEARANCE,
    { theme: 'dark', textSize: 'large', rows: 'compact', accent: 'blue' },
    { theme: 'light', textSize: 'larger', rows: 'comfortable', accent: 'pink' },
    { theme: 'system', textSize: 'normal', rows: 'compact', accent: 'purple' },
  ];

  it.each(cases)('sets what appearance.ts sets: %o', (appearance) => {
    const stored: Record<string, string> = {};
    for (const key of Object.keys(APPEARANCE_KEYS) as (keyof Appearance)[]) {
      if (!isDefault(key, appearance[key])) stored[APPEARANCE_KEYS[key]] = appearance[key];
    }
    const expected = Object.fromEntries(
      Object.entries(rootAttributes(appearance)).filter(([, value]) => value !== null),
    );
    expect(run(stored)).toEqual(expected);
  });

  it('ignores values it does not know and survives blocked storage', () => {
    expect(run({ [APPEARANCE_KEYS.theme]: 'purple' })).toEqual({});
    const blocked = {
      get localStorage(): never {
        throw new Error('blocked');
      },
    };
    expect(() => new Function('window', 'document', script)(blocked, {})).not.toThrow();
  });
});

describe('the accents in app.css', () => {
  const css = readFileSync('src/app.css', 'utf8');

  function luminance(hex: string): number {
    const channel = (i: number) => {
      const v = parseInt(hex.slice(1 + i * 2, 3 + i * 2), 16) / 255;
      return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
    };
    return 0.2126 * channel(0) + 0.7152 * channel(1) + 0.0722 * channel(2);
  }
  const contrast = (a: string, b: string) => {
    const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
  };

  const lightBg = /:root \{[^}]*--bg: (#[0-9a-f]{6})/.exec(css)![1];
  const darkBg = /:root\[data-theme='dark'\] \{[^}]*--bg: (#[0-9a-f]{6})/.exec(css)![1];

  it('has a light and a dark value for each accent, readable on the page', () => {
    for (const accent of ACCENTS.filter((a) => a !== 'green')) {
      const light = new RegExp(
        `:root\\[data-accent='${accent}'\\] \\{\\s*--accent: (#[0-9a-f]{6})`,
      ).exec(css);
      const dark = new RegExp(
        `:root\\[data-theme='dark'\\]\\[data-accent='${accent}'\\] \\{\\s*--accent: (#[0-9a-f]{6})`,
      ).exec(css);
      expect(light, `${accent} light`).not.toBeNull();
      expect(dark, `${accent} dark`).not.toBeNull();
      expect(light![1], `${accent} swatch`).toBe(ACCENT_SWATCHES[accent]);
      expect(contrast(light![1], lightBg), `${accent} on light`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(dark![1], darkBg), `${accent} on dark`).toBeGreaterThanOrEqual(4.5);
    }
  });

  it('uses the same swatch for green as the root value', () => {
    expect(/:root \{[^}]*--accent: (#[0-9a-f]{6})/.exec(css)![1]).toBe(ACCENT_SWATCHES.green);
  });
});

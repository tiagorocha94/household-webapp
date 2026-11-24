// The appearance choices as reactive state, kept in this browser's local storage.
import { clearLanguageChoice } from './i18n/index.svelte';
import {
  APPEARANCE_KEYS,
  applyAppearance,
  DEFAULT_APPEARANCE,
  isDefault,
  parseAppearance,
  type Appearance,
} from './appearance';

function readKey(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeKey(key: string, value: string | null): void {
  try {
    if (value === null) window.localStorage.removeItem(key);
    else window.localStorage.setItem(key, value);
  } catch {
    // The choice then lasts until the page is closed.
  }
}

const state = $state<Appearance>(
  typeof window === 'undefined' ? { ...DEFAULT_APPEARANCE } : parseAppearance(readKey),
);

if (typeof document !== 'undefined') applyAppearance(state, document.documentElement);

/** The current choices. */
export const appearance: Readonly<Appearance> = state;

/** Changes one choice, applies it at once and remembers it (the default is remembered as nothing). */
export function setAppearance<K extends keyof Appearance>(key: K, value: Appearance[K]): void {
  state[key] = value;
  if (typeof document !== 'undefined') applyAppearance(state, document.documentElement);
  writeKey(APPEARANCE_KEYS[key], isDefault(key, value) ? null : value);
}

/** Back to the defaults for everything: the theme and language follow the system again. */
export function resetSettings(): void {
  for (const key of Object.keys(APPEARANCE_KEYS) as (keyof Appearance)[]) {
    setAppearance(key, DEFAULT_APPEARANCE[key]);
  }
  clearLanguageChoice();
}

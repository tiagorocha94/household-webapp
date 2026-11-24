// A small translation layer: English and Portuguese (Portugal), no library. The texts are plain
// objects and `t` fills in `{name}` placeholders. The language is reactive, so a text shown with
// `t` changes when the person switches it.
import { categoryNamesPt, groupNamesPt } from './category-names';
import { en, type MessageKey } from './en';
import { pt } from './pt';

export type { MessageKey } from './en';
export type Locale = 'en' | 'pt';

const messages: Record<Locale, Record<MessageKey, string>> = { en, pt };

/** The tag Intl uses for each language. */
export const LOCALE_TAGS: Record<Locale, string> = { en: 'en-GB', pt: 'pt-PT' };

const STORAGE_KEY = 'household.language';

/**
 * Picks the language from the browser's list of preferred languages (for example
 * `["pt-PT", "en-GB"]`): the first one that is Portuguese or English decides, and anything else
 * falls back to English.
 */
export function detectLocale(languages: readonly string[]): Locale {
  for (const tag of languages) {
    const primary = tag.trim().toLowerCase().split(/[-_]/)[0];
    if (primary === 'pt') return 'pt';
    if (primary === 'en') return 'en';
  }
  return 'en';
}

/** The language the person chose, when it is a valid one. */
export function chosenLocale(stored: string | null): Locale | null {
  return stored === 'en' || stored === 'pt' ? stored : null;
}

function storedChoice(): Locale | null {
  if (typeof window === 'undefined') return null;
  try {
    return chosenLocale(window.localStorage.getItem(STORAGE_KEY));
  } catch {
    return null;
  }
}

function startingLocale(): Locale {
  if (typeof window === 'undefined') return 'en';
  try {
    const chosen = chosenLocale(window.localStorage.getItem(STORAGE_KEY));
    if (chosen) return chosen;
  } catch {
    // Storage can be blocked; the browser's language decides then.
  }
  const languages = navigator.languages?.length ? navigator.languages : [navigator.language ?? ''];
  return detectLocale(languages);
}

const state = $state<{ locale: Locale; chosen: boolean }>({
  locale: startingLocale(),
  chosen: storedChoice() !== null,
});

function applyToPage(locale: Locale): void {
  if (typeof document === 'undefined') return;
  document.documentElement.lang = LOCALE_TAGS[locale];
}
applyToPage(state.locale);

export function getLocale(): Locale {
  return state.locale;
}

/** Sets the language of every text shown from now on, and remembers the choice in this browser. */
export function setLocale(locale: Locale): void {
  state.locale = locale;
  state.chosen = true;
  applyToPage(locale);
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, locale);
  } catch {
    // The choice then lasts until the page is closed.
  }
}

/** True when the person chose a language, false while the browser's language decides. */
export function hasLanguageChoice(): boolean {
  return state.chosen;
}

/** Goes back to the browser's language and forgets the choice. */
export function clearLanguageChoice(): void {
  if (typeof window !== 'undefined') {
    try {
      window.localStorage.removeItem(STORAGE_KEY);
    } catch {
      // Nothing to forget then.
    }
  }
  const languages =
    typeof navigator === 'undefined'
      ? []
      : navigator.languages?.length
        ? navigator.languages
        : [navigator.language ?? ''];
  state.locale = detectLocale(languages);
  state.chosen = false;
  applyToPage(state.locale);
}

/** The tag for Intl formatting in the current language. */
export function localeTag(): string {
  return LOCALE_TAGS[state.locale];
}

/** Fills the `{name}` placeholders of a text. */
export function fill(text: string, params: Record<string, string | number>): string {
  return text.replace(/\{(\w+)\}/g, (whole, name: string) =>
    name in params ? String(params[name]) : whole,
  );
}

/** The text for a key in the current language, with `{name}` placeholders filled in. */
export function t(key: MessageKey, params: Record<string, string | number> = {}): string {
  return fill(messages[state.locale][key], params);
}

function named(table: Readonly<Record<string, string>>, name: string): string {
  if (state.locale !== 'pt' || !Object.prototype.hasOwnProperty.call(table, name)) return name;
  return table[name];
}

/** What to show for a category name; a name without a translation is shown as it is. */
export function categoryLabel(name: string): string {
  return named(categoryNamesPt, name);
}

/** What to show for a budget group name; a name without a translation is shown as it is. */
export function groupLabel(name: string): string {
  return named(groupNamesPt, name);
}

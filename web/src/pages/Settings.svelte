<script lang="ts">
  import Segmented from '../components/Segmented.svelte';
  import {
    ACCENTS,
    ROW_DENSITIES,
    TEXT_SIZES,
    THEMES,
    type Accent,
    type RowDensity,
    type TextSize,
    type Theme,
  } from '../lib/appearance';
  import { appearance, resetSettings, setAppearance } from '../lib/appearance.svelte';
  import { ACCENT_SWATCHES } from '../lib/settings';
  import {
    clearLanguageChoice,
    getLocale,
    hasLanguageChoice,
    setLocale,
    t,
    type Locale,
  } from '../lib/i18n/index.svelte';

  type LanguageChoice = 'system' | Locale;
  const language = $derived<LanguageChoice>(hasLanguageChoice() ? getLocale() : 'system');

  function chooseLanguage(choice: LanguageChoice) {
    if (choice === 'system') clearLanguageChoice();
    else setLocale(choice);
  }
</script>

<h1>{t('shell.settings.title')}</h1>
<p class="muted">{t('shell.settings.help')}</p>

<h2>{t('shell.settings.appearance')}</h2>
<div class="rows">
  <div class="row">
    <div class="what">
      <b>{t('shell.settings.theme')}</b>
      <span class="muted">{t('shell.settings.themeHelp')}</span>
    </div>
    <Segmented
      label={t('shell.settings.theme')}
      value={appearance.theme}
      options={THEMES.map((value: Theme) => ({ value, label: t(`shell.settings.theme.${value}`) }))}
      onchange={(value) => setAppearance('theme', value)}
    />
  </div>
  <div class="row">
    <div class="what">
      <b>{t('shell.language.label')}</b>
      <span class="muted">{t('shell.settings.languageHelp')}</span>
    </div>
    <Segmented
      label={t('shell.language.label')}
      value={language}
      options={[
        { value: 'system', label: t('shell.settings.theme.system') },
        { value: 'en', label: 'English', lang: 'en' },
        { value: 'pt', label: 'Português', lang: 'pt' },
      ]}
      onchange={chooseLanguage}
    />
  </div>
  <div class="row">
    <div class="what">
      <b>{t('shell.settings.textSize')}</b>
      <span class="muted">{t('shell.settings.textSizeHelp')}</span>
    </div>
    <Segmented
      label={t('shell.settings.textSize')}
      value={appearance.textSize}
      options={TEXT_SIZES.map((value: TextSize) => ({
        value,
        label: t(`shell.settings.textSize.${value}`),
      }))}
      onchange={(value) => setAppearance('textSize', value)}
    />
  </div>
  <div class="row">
    <div class="what">
      <b>{t('shell.settings.rows')}</b>
      <span class="muted">{t('shell.settings.rowsHelp')}</span>
    </div>
    <Segmented
      label={t('shell.settings.rows')}
      value={appearance.rows}
      options={ROW_DENSITIES.map((value: RowDensity) => ({
        value,
        label: t(`shell.settings.rows.${value}`),
      }))}
      onchange={(value) => setAppearance('rows', value)}
    />
  </div>
  <div class="row">
    <div class="what">
      <b>{t('shell.settings.accent')}</b>
      <span class="muted">{t('shell.settings.accentHelp')}</span>
    </div>
    <Segmented
      label={t('shell.settings.accent')}
      value={appearance.accent}
      options={ACCENTS.map((value: Accent) => ({
        value,
        label: t(`shell.settings.accent.${value}`),
        swatch: ACCENT_SWATCHES[value],
      }))}
      onchange={(value) => setAppearance('accent', value)}
    />
  </div>
</div>

<p class="reset">
  <button type="button" onclick={resetSettings}>{t('shell.settings.reset')}</button>
  <span class="muted">{t('shell.settings.resetHelp')}</span>
</p>

<style>
  .rows {
    border-top: 1px solid var(--ink);
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem 1.5rem;
    padding: 0.9rem 0;
    border-bottom: 1px solid var(--rule);
  }
  .what {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
    flex: 1 1 16rem;
  }
  .what .muted {
    font-size: 0.85rem;
  }
  .reset {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem 1rem;
    margin-top: 1.25rem;
  }
  .reset button {
    min-height: 2.75rem;
    padding: 0 1rem;
    border: 1px solid var(--rule);
    border-radius: 8px;
    background: transparent;
    color: var(--ink);
    font: inherit;
    cursor: pointer;
  }
  .reset .muted {
    font-size: 0.85rem;
  }
</style>

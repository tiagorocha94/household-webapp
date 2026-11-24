<script lang="ts">
  import { getLocale, setLocale, t, type Locale } from '../lib/i18n/index.svelte';
  import Segmented from './Segmented.svelte';

  const options: { value: Locale; label: string; lang: Locale }[] = [
    { value: 'en', label: 'EN', lang: 'en' },
    { value: 'pt', label: 'PT', lang: 'pt' },
  ];
</script>

<Segmented
  label={t('shell.language.label')}
  {options}
  value={getLocale()}
  onchange={(locale) => setLocale(locale)}
/>

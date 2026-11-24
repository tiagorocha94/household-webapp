<script lang="ts">
  import { t } from '../lib/i18n/index.svelte';
  import { href } from '../lib/router.svelte';
</script>

<h1>{t('shell.notFound.title')}</h1>
<p class="muted">
  {t('shell.notFound.text')} <a href={href('/')}>{t('shell.notFound.link')}</a>.
</p>

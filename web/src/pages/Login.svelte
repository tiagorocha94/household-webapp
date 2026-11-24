<script lang="ts">
  import { ApiError } from '../api/client';
  import LanguageSwitch from '../components/LanguageSwitch.svelte';
  import { t } from '../lib/i18n/index.svelte';
  import { navigate } from '../lib/router.svelte';
  import { logIn } from '../lib/session.svelte';

  let username = $state('');
  let password = $state('');
  let message = $state('');
  let busy = $state(false);

  function explain(error: unknown): string {
    if (!(error instanceof ApiError)) return t('shell.error.generic');
    switch (error.status) {
      case 0:
        return t('shell.error.unreachable');
      case 400:
        return t('shell.login.enterBoth');
      case 401:
        return t('shell.login.wrong');
      case 429: {
        const minutes = Math.max(1, Math.ceil((error.retryAfterSeconds ?? 60) / 60));
        return t(minutes === 1 ? 'shell.login.tooManyOne' : 'shell.login.tooManyOther', {
          minutes,
        });
      }
      default:
        return t('shell.error.generic');
    }
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    message = '';
    try {
      await logIn(username, password);
      navigate('/');
    } catch (error) {
      message = explain(error);
    } finally {
      busy = false;
    }
  }
</script>

<main class="login">
  <div class="corner"><LanguageSwitch /></div>
  <h1>{t('shell.login.title')}</h1>
  <form onsubmit={submit}>
    <label>
      {t('shell.login.username')}
      <input bind:value={username} autocomplete="username" required />
    </label>
    <label>
      {t('shell.login.password')}
      <input bind:value={password} type="password" autocomplete="current-password" required />
    </label>
    {#if message}
      <p class="form-error" role="alert">{message}</p>
    {/if}
    <button type="submit" class="primary" disabled={busy}>{t('shell.login.submit')}</button>
  </form>
</main>

<style>
  .login {
    max-width: 22rem;
    margin: 12vh auto 0;
    padding: 0 1rem;
  }
  .corner {
    display: flex;
    justify-content: flex-end;
  }
  form {
    display: grid;
    gap: 1rem;
    margin-top: 1.5rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    font-size: 0.9rem;
    color: var(--muted);
  }
  .form-error {
    margin: 0;
    color: var(--out);
  }
</style>

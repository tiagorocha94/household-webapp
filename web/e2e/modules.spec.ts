import { expect, test, type Page } from '@playwright/test';

// The modules each project runs (playwright.config.ts), by project name.
const RUNNING: Record<string, string[]> = {
  all: ['cashflow', 'goals', 'savings', 'credits', 'investments'],
  cashflow: ['cashflow'],
  'goals+savings': ['goals', 'savings'],
};
const ALL = RUNNING.all;

async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel('Username').fill('alex');
  await page.getByLabel('Password').fill('demo');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible();
}

test.beforeEach(async ({ page }) => {
  await signIn(page);
});

test('the menu shows only the modules that run', async ({ page }, info) => {
  const running = RUNNING[info.project.name];
  const menu = page.getByRole('navigation', { name: 'Main' });
  await expect(menu.locator('a[href="#/"]')).toBeVisible();
  await expect(menu.locator('a[href="#/net-worth"]')).toBeVisible();
  for (const module of ALL) {
    const item = menu.locator(`a[href="#/${module}"]`);
    if (running.includes(module)) await expect(item).toBeVisible();
    else await expect(item).toHaveCount(0);
  }
});

test('the server says which modules run', async ({ page }, info) => {
  const response = await page.request.get('/api/v1/info');
  const body = await response.json();
  expect(body.modules).toEqual(RUNNING[info.project.name]);
});

test('Home shows only the sections of the modules that run', async ({ page }, info) => {
  const running = RUNNING[info.project.name];
  await page.goto('/#/');
  // Cashflow is the first section, so once it shows (or is known to be left out) the page has loaded.
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  for (const module of ALL) {
    const section = page.locator(`section[aria-labelledby="${module}"]`);
    if (running.includes(module)) await expect(section).toBeVisible();
    else await expect(section).toHaveCount(0);
  }
});

test('a page of a module that does not run is not found, and the others open', async ({
  page,
}, info) => {
  const running = RUNNING[info.project.name];
  for (const module of ALL) {
    await page.goto(`/#/${module}`);
    const notFound = page.getByRole('heading', { name: 'Page not found' });
    if (running.includes(module)) await expect(notFound).toHaveCount(0);
    else await expect(notFound).toBeVisible();
  }
});

test('Net worth opens whatever runs', async ({ page }) => {
  await page.goto('/#/net-worth');
  await expect(page.getByRole('heading', { name: 'Page not found' })).toHaveCount(0);
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
});

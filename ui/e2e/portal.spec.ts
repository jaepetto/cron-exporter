import { expect, test } from '@playwright/test';

test('complete job lifecycle and deep-link reload', async ({ page }, testInfo) => {
  const jobName = `nightly-backup-${testInfo.project.name}-${testInfo.retry}`;
  await page.goto('/dashboard/jobs');
  await expect(page.getByRole('heading', { name: 'Jobs', exact: true })).toBeVisible();

  await page.getByRole('link', { name: 'New job' }).first().click();
  await page.getByLabel('Job name').fill(jobName);
  await page.getByLabel('Host').fill('db01');
  await page.getByLabel('Failure threshold (seconds)').fill('1800');
  await page.getByLabel('Labels (JSON)').fill('{"env":"e2e","team":"platform"}');
  await page.getByRole('button', { name: 'Save job' }).click();

  await expect(page.getByRole('heading', { name: jobName })).toBeVisible();
  await expect(page.getByText('db01')).toBeVisible();
  await page.reload();
  await expect(page.getByRole('heading', { name: jobName })).toBeVisible();

  await page.getByRole('button', { name: 'Maintenance' }).click();
  await expect(page.getByText('maintenance', { exact: true })).toBeVisible();

  await page.getByRole('link', { name: 'Edit' }).click();
  await page.getByLabel('Failure threshold (seconds)').fill('7200');
  await page.getByRole('button', { name: 'Save job' }).click();
  await expect(page.getByText('2h', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByRole('alertdialog')).toBeVisible();
  await page.getByRole('button', { name: 'Delete job' }).click();
  await expect(page.getByRole('heading', { name: 'Jobs', exact: true })).toBeVisible();
  await expect(page.getByText(jobName)).toHaveCount(0);
});

test('search, live invalidation, and theme persistence', async ({ page }, testInfo) => {
  const jobName = `live-search-${testInfo.project.name}-${testInfo.retry}`;
  await page.goto('/dashboard/jobs');
  await page.getByPlaceholder('Search name, host, or label').fill(jobName);
  await expect(page.getByRole('heading', { name: 'No jobs found' })).toBeVisible();
  const createResponse = await page.evaluate(async (newJobName) => {
    const response = await fetch('/dashboard/api/jobs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ job_name: newJobName, host: 'web01', labels: { env: 'live' } }),
    });
    return response.status;
  }, jobName);
  expect(createResponse).toBe(201);
  await expect(page.getByText(jobName).first()).toBeVisible();

  await page.getByPlaceholder('Search name, host, or label').fill('does-not-exist');
  await expect(page.getByRole('heading', { name: 'No jobs found' })).toBeVisible();
  await page.getByPlaceholder('Search name, host, or label').fill(jobName);
  await expect(page.getByText(jobName).first()).toBeVisible();

  await page.getByRole('combobox', { name: 'Theme' }).click();
  await page.getByRole('option', { name: 'Dark' }).click();
  await expect(page.locator('html')).toHaveClass(/dark/);
  await page.reload();
  await expect(page.locator('html')).toHaveClass(/dark/);
});

test('mobile layout does not create horizontal page overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/dashboard/jobs');
  await expect(page.getByRole('heading', { name: 'Jobs', exact: true })).toBeVisible();

  const dimensions = await page.evaluate(() => ({ width: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.width);
  await expect(page.getByRole('navigation', { name: 'Primary navigation' })).toBeVisible();
});

test('dashboard API rejects missing credentials', async ({ baseURL }) => {
  const response = await fetch(`${baseURL}/dashboard/api/config`);
  expect(response.status).toBe(401);
  expect(response.headers.get('www-authenticate')).toContain('Basic');
});

test('large job collections remain bounded and responsive', async ({ browserName, page }, testInfo) => {
  test.skip(browserName !== 'chromium', 'Large dataset and screenshot verification runs once in Chromium');
  await page.goto('/dashboard/jobs');
  const prefix = `scale-${testInfo.retry}`;
  const result = await page.evaluate(async ({ jobPrefix, count }) => {
    for (let index = 0; index < count; index += 1) {
      const response = await fetch('/dashboard/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ job_name: `${jobPrefix}-${index}`, host: `host-${index % 20}`, api_key: `cm_${jobPrefix}_${index}`, labels: { env: 'scale' } }),
      });
      if (response.status !== 201 && response.status !== 409) throw new Error(`create failed: ${response.status}`);
    }
    const started = performance.now();
    const response = await fetch('/dashboard/api/jobs?page=20&page_size=25');
    const data = await response.json() as { jobs: unknown[]; total_count: number };
    return { duration: performance.now() - started, pageSize: data.jobs.length, total: data.total_count };
  }, { jobPrefix: prefix, count: 1000 });

  expect(result.pageSize).toBe(25);
  expect(result.total).toBeGreaterThanOrEqual(1000);
  expect(result.duration).toBeLessThan(2000);
  await page.reload();
  await expect(page.locator('.job-table tbody tr')).toHaveCount(25);
  await page.screenshot({ path: testInfo.outputPath('large-desktop.png'), fullPage: true });
  const desktop = await page.evaluate(() => ({ width: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  expect(desktop.scrollWidth).toBeLessThanOrEqual(desktop.width);

  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: testInfo.outputPath('large-mobile.png'), fullPage: true });
  const mobile = await page.evaluate(() => ({ width: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  expect(mobile.scrollWidth).toBeLessThanOrEqual(mobile.width);
});

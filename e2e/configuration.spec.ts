import { test, expect } from '@grafana/plugin-e2e';

/**
 * Configuration-page smoke tests for the Google Managed Service for
 * Prometheus plugin. Driven by @grafana/plugin-e2e against the
 * provisioned local data sources in provisioning/datasources/.
 *
 * These tests cover the surfaces unique to this plugin (the Google auth
 * section, the conflict warning, the secret-handling flow). They do NOT
 * re-test the upstream PromSettings / Alerting / AdvancedHttp blocks —
 * those are owned by @grafana/prometheus and tested upstream.
 */
test.describe('Google Cloud authentication section', () => {
  test('renders the heading with the selector defaulting to None', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml', name: 'Local Prom' });
    await createDataSourceConfigPage({ type: ds.type });

    await expect(page.getByRole('heading', { name: 'Google Cloud authentication' })).toBeVisible();
    await expect(page.getByText('None').first()).toBeVisible();
    await expect(page.getByLabel('Google Cloud service account JSON')).toHaveCount(0);
  });

  test('reveals the JSON textarea when Service Account JSON is selected', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml', name: 'Local Prom' });
    await createDataSourceConfigPage({ type: ds.type });

    await page.getByLabel('Google Cloud authentication').click();
    await page.getByRole('option', { name: 'Service Account JSON' }).click();

    await expect(page.getByLabel('Google Cloud service account JSON')).toBeVisible();
  });

  test('round-trips the auth type via Save & test', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml', name: 'Local Prom' });
    const configPage = await createDataSourceConfigPage({ type: ds.type });

    await page.getByLabel('Google Cloud authentication').click();
    await page.getByRole('option', { name: 'Google Cloud ADC' }).click();
    await configPage.saveAndTest({ skipDataSourceCheck: true });

    await page.reload();
    await expect(page.getByText('Google Cloud ADC').first()).toBeVisible();
  });

  test('Basic auth + Google ADC shows the conflict warning', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml', name: 'Local Prom' });
    await createDataSourceConfigPage({ type: ds.type });

    // Basic auth is selected in the upstream Authentication-methods picker.
    await page.locator('#auth-method-select').click();
    await page.getByRole('option', { name: /Basic auth/i }).click();
    await page.getByLabel('Google Cloud authentication').click();
    await page.getByRole('option', { name: 'Google Cloud ADC' }).click();

    await expect(page.getByText(/Google Cloud authentication will override .*Basic auth/)).toBeVisible();
  });

  test('Service Account JSON secret is never returned over the API', async ({ readProvisionedDataSource, request }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml', name: 'GMP (Service Account)' });
    // Provisioned via secureJsonData; the API never echoes the value back.
    const r = await request.get(`/api/datasources/uid/${ds.uid}`);
    const body = await r.json();
    expect(body.secureJsonFields?.googleServiceAccountJson).toBe(true);
    expect(JSON.stringify(body)).not.toContain('-----BEGIN PRIVATE KEY-----');
  });
});

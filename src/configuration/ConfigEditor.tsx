import { type DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { t, Trans } from '@grafana/i18n';
import { AdvancedHttpSettings, ConfigSection, DataSourceDescription } from '@grafana/plugin-ui';
import {
  AlertingSettingsOverhaul,
  DataSourceHttpSettingsOverhaul,
  overhaulStyles,
  PromSettings,
} from '@grafana/prometheus';
import { config } from '@grafana/runtime';
import { Alert, useTheme2 } from '@grafana/ui';

import { type DataSourceOptions, type DataSourceSecureJsonData } from './DataSourceOptions';
import { GoogleAuthSection } from './GoogleAuthSection';

export type Props = DataSourcePluginOptionsEditorProps<DataSourceOptions, DataSourceSecureJsonData>;

export const ConfigEditor = (props: Props) => {
  const { options, onOptionsChange } = props;
  const theme = useTheme2();
  const styles = overhaulStyles(theme);

  return (
    <>
      {options.access === 'direct' && (
        <Alert title={t('configuration.config-editor.title-error', 'Error')} severity="error">
          <Trans i18nKey="configuration.config-editor.browser-access-mode-error">
            Browser access mode in the Google Managed Service for Prometheus data source is no longer available. Switch
            to server access mode.
          </Trans>
        </Alert>
      )}

      <DataSourceDescription
        dataSourceName="Google Managed Service for Prometheus"
        docsLink="https://github.com/binboum/binboum-googleprometheus-datasource#readme"
      />
      <hr className={`${styles.hrTopSpace} ${styles.hrBottomSpace}`} />

      <DataSourceHttpSettingsOverhaul
        options={options}
        onOptionsChange={onOptionsChange}
        secureSocksDSProxyEnabled={config.secureSocksDSProxyEnabled}
      />

      <GoogleAuthSection options={options} onOptionsChange={onOptionsChange} />

      <hr />
      <ConfigSection
        className={styles.advancedSettings}
        title={t('configuration.config-editor.title-advanced-settings', 'Advanced settings')}
        description={t(
          'configuration.config-editor.description-advanced-settings',
          'Additional settings are optional settings that can be configured for more control over your data source.'
        )}
      >
        <AdvancedHttpSettings
          className={styles.advancedHTTPSettingsMargin}
          config={options}
          onChange={onOptionsChange}
        />
        <AlertingSettingsOverhaul<DataSourceOptions> options={options} onOptionsChange={onOptionsChange} />
        <PromSettings options={options} onOptionsChange={onOptionsChange} hidePrometheusTypeVersion hideExemplars />
      </ConfigSection>
    </>
  );
};

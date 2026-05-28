import { type DataSourceSettings, type SelectableValue } from '@grafana/data';
import { ConfigSection } from '@grafana/plugin-ui';
import { Alert, Field, SecretTextArea, Select, Stack } from '@grafana/ui';
import { type ChangeEvent } from 'react';

import { type DataSourceOptions, type DataSourceSecureJsonData } from './DataSourceOptions';

// Wire values written to jsonData.googleAuthType. Empty string means disabled
// (the backend treats absent and "" as the same no-op).
export const GoogleAuthType = {
  None: '',
  ADC: 'adc',
  ServiceAccountJSON: 'serviceAccountJson',
} as const;
export type GoogleAuthTypeValue = (typeof GoogleAuthType)[keyof typeof GoogleAuthType];

const options: Array<SelectableValue<GoogleAuthTypeValue>> = [
  { label: 'None', value: GoogleAuthType.None },
  {
    label: 'Google Cloud ADC',
    value: GoogleAuthType.ADC,
    description: 'Use Application Default Credentials (metadata server or GOOGLE_APPLICATION_CREDENTIALS)',
  },
  {
    label: 'Service Account JSON',
    value: GoogleAuthType.ServiceAccountJSON,
    description: 'Use a service account JSON key',
  },
];

type Props = {
  options: DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>;
  onOptionsChange: (options: DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>) => void;
};

/** Renders the additive Google Cloud authentication section. */
export const GoogleAuthSection = ({ options: dsOptions, onOptionsChange }: Props) => {
  const jsonData = dsOptions.jsonData;
  const secureJsonData = (dsOptions.secureJsonData ?? {}) as DataSourceSecureJsonData;
  const secureJsonFields = dsOptions.secureJsonFields ?? {};
  const googleAuthType: GoogleAuthTypeValue = jsonData.googleAuthType ?? GoogleAuthType.None;
  const googleAuthEnabled = googleAuthType !== GoogleAuthType.None;
  const conflicts = collectConflicts(dsOptions);

  const onAuthTypeChange = (selected: SelectableValue<GoogleAuthTypeValue> | null) => {
    onOptionsChange({
      ...dsOptions,
      jsonData: { ...dsOptions.jsonData, googleAuthType: selected?.value ?? GoogleAuthType.None },
    });
  };

  const onJsonKeyChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onOptionsChange({
      ...dsOptions,
      secureJsonData: { ...(dsOptions.secureJsonData ?? {}), googleServiceAccountJson: event.currentTarget.value },
    });
  };

  const onJsonKeyReset = () => {
    onOptionsChange({
      ...dsOptions,
      secureJsonFields: { ...secureJsonFields, googleServiceAccountJson: false },
      secureJsonData: { ...(dsOptions.secureJsonData ?? {}), googleServiceAccountJson: '' },
    });
  };

  return (
    <ConfigSection title="Google Cloud authentication">
      <Stack direction="column" gap={2}>
        <Field label="Authentication type" noMargin>
          <Select
            aria-label="Google Cloud authentication"
            inputId="google-prometheus-auth-type"
            width={40}
            options={options}
            value={options.find((o) => o.value === googleAuthType) ?? options[0]}
            onChange={onAuthTypeChange}
          />
        </Field>

        {googleAuthEnabled && conflicts.length > 0 && (
          <Alert severity="warning" title="Conflicting authentication">
            Google Cloud authentication will override {conflicts.join(', ')} for outgoing requests.
          </Alert>
        )}

        {googleAuthType === GoogleAuthType.ServiceAccountJSON && (
          <Field
            noMargin
            label="Service Account JSON key"
            description="Paste the contents of a Google Cloud service account JSON key. The value is encrypted at rest."
          >
            <SecretTextArea
              aria-label="Google Cloud service account JSON"
              id="google-prometheus-service-account-json"
              placeholder='{"type":"service_account", ...}'
              cols={45}
              rows={7}
              isConfigured={secureJsonFields.googleServiceAccountJson === true}
              value={secureJsonData.googleServiceAccountJson ?? ''}
              onChange={onJsonKeyChange}
              onReset={onJsonKeyReset}
            />
          </Field>
        )}
      </Stack>
    </ConfigSection>
  );
};

/** Returns the human-readable names of auth flags that Google auth overrides. */
function collectConflicts(dsOptions: DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>): string[] {
  const out: string[] = [];
  if (dsOptions.basicAuth) {
    out.push('Basic auth');
  }
  if (dsOptions.jsonData.oauthPassThru) {
    out.push('Forward OAuth Identity');
  }
  return out;
}

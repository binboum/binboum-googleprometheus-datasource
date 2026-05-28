import { type PromOptions } from '@grafana/prometheus';

import { type GoogleAuthTypeValue } from './GoogleAuthSection';

/**
 * DataSourceOptions extends the upstream PromOptions with the Google-auth
 * fields persisted in jsonData. secureJsonData is typed separately so the
 * frontend can never accidentally serialise the secret into jsonData.
 */
export type DataSourceOptions = PromOptions & {
  /**
   * Selects the Google-auth mode used by the backend's HTTP middleware.
   *   - undefined / ''      : disabled (no Google middleware appended)
   *   - 'adc'               : Application Default Credentials
   *   - 'serviceAccountJson': pasted service-account JSON key
   */
  googleAuthType?: GoogleAuthTypeValue;
};

export type DataSourceSecureJsonData = {
  /** Raw service-account JSON key body. Persisted encrypted, never echoed back. */
  googleServiceAccountJson?: string;
};

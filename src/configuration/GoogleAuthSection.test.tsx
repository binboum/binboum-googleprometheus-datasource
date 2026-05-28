import '@testing-library/jest-dom';

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { select } from 'react-select-event';

import { type DataSourceSettings } from '@grafana/data';

import { type DataSourceOptions, type DataSourceSecureJsonData } from './DataSourceOptions';
import { GoogleAuthSection, GoogleAuthType } from './GoogleAuthSection';

function defaultOptions(
  overrides: Partial<DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>> = {}
): DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData> {
  return {
    jsonData: {},
    secureJsonData: {},
    secureJsonFields: {},
    access: 'proxy',
    basicAuth: false,
    withCredentials: false,
    ...overrides,
  } as DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>;
}

function renderSection(overrides: Partial<DataSourceSettings<DataSourceOptions, DataSourceSecureJsonData>> = {}) {
  const onOptionsChange = jest.fn();
  render(<GoogleAuthSection options={defaultOptions(overrides)} onOptionsChange={onOptionsChange} />);
  return { onOptionsChange };
}

async function selectGoogleAuthOption(label: string) {
  const input = screen.getByLabelText('Google Cloud authentication');
  await waitFor(() => select(input, label, { container: document.body }));
}

describe('GoogleAuthSection', () => {
  it('renders the heading with the selector defaulting to None', () => {
    renderSection();
    expect(screen.getByRole('heading', { name: 'Google Cloud authentication' })).toBeInTheDocument();
    expect(screen.getByText('None')).toBeInTheDocument();
    expect(screen.queryByLabelText('Google Cloud service account JSON')).not.toBeInTheDocument();
  });

  it('persists ADC selection without revealing the JSON textarea', async () => {
    const { onOptionsChange } = renderSection();
    await selectGoogleAuthOption('Google Cloud ADC');
    expect(onOptionsChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({ googleAuthType: GoogleAuthType.ADC }),
      })
    );
    expect(screen.queryByLabelText('Google Cloud service account JSON')).not.toBeInTheDocument();
  });

  it('reveals the JSON textarea when Service Account JSON is the selected type', () => {
    renderSection({ jsonData: { googleAuthType: GoogleAuthType.ServiceAccountJSON } });
    expect(screen.getByLabelText('Google Cloud service account JSON')).toBeInTheDocument();
  });

  it('persists Service Account JSON selection', async () => {
    const { onOptionsChange } = renderSection();
    await selectGoogleAuthOption('Service Account JSON');
    expect(onOptionsChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({ googleAuthType: GoogleAuthType.ServiceAccountJSON }),
      })
    );
  });

  it('writes typed JSON to secureJsonData.googleServiceAccountJson', async () => {
    const { onOptionsChange } = renderSection({ jsonData: { googleAuthType: GoogleAuthType.ServiceAccountJSON } });
    const textarea = screen.getByLabelText('Google Cloud service account JSON');
    // user-event treats `{` as a key-descriptor delimiter — double it to type a literal `{`.
    await userEvent.type(textarea, '{{');
    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        secureJsonData: expect.objectContaining({ googleServiceAccountJson: '{' }),
      })
    );
  });

  it('shows the Configured / Reset state when the secret is persisted', () => {
    renderSection({
      jsonData: { googleAuthType: GoogleAuthType.ServiceAccountJSON },
      secureJsonFields: { googleServiceAccountJson: true },
    });
    expect(screen.getByRole('button', { name: /reset/i })).toBeInTheDocument();
  });

  it('Reset clears the secret and lowers secureJsonFields', async () => {
    const { onOptionsChange } = renderSection({
      jsonData: { googleAuthType: GoogleAuthType.ServiceAccountJSON },
      secureJsonFields: { googleServiceAccountJson: true },
    });
    await userEvent.click(screen.getByRole('button', { name: /reset/i }));
    expect(onOptionsChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        secureJsonFields: expect.objectContaining({ googleServiceAccountJson: false }),
        secureJsonData: expect.objectContaining({ googleServiceAccountJson: '' }),
      })
    );
  });

  it('warns when ADC is on and Basic auth is also configured', () => {
    renderSection({ basicAuth: true, jsonData: { googleAuthType: GoogleAuthType.ADC } });
    expect(screen.getByText(/Google Cloud authentication will override.*Basic auth/i)).toBeInTheDocument();
  });

  it('warns when ADC is on and oauthPassThru is also configured', () => {
    renderSection({ jsonData: { googleAuthType: GoogleAuthType.ADC, oauthPassThru: true } });
    expect(screen.getByText(/Google Cloud authentication will override.*Forward OAuth Identity/i)).toBeInTheDocument();
  });

  it('does not warn when only Google auth is configured', () => {
    renderSection({ jsonData: { googleAuthType: GoogleAuthType.ADC } });
    expect(screen.queryByText(/Google Cloud authentication will override/i)).not.toBeInTheDocument();
  });

  it('does not warn when Google auth is disabled, even if other modes are set', () => {
    renderSection({ basicAuth: true, jsonData: { oauthPassThru: true } });
    expect(screen.queryByText(/Google Cloud authentication will override/i)).not.toBeInTheDocument();
  });
});

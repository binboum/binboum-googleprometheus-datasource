import path from 'node:path';

import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';

import grafanaConfig, { type Env } from './.config/webpack/webpack.config';

// Force @grafana/i18n and react-i18next to a single canonical path. The
// @grafana/prometheus dependency installs its own nested copies; only one gets
// initialized by initPluginTranslations(), causing t() errors at runtime if
// the wrong instance is reached. Same workaround the AWS plugin applies.
// The vendored swc-loader config defaults to the classic JSX runtime, but the
// source uses the automatic runtime (tsconfig jsx: react-jsx) and never imports
// React. Without this the bundle emits React.createElement and crashes at load
// with "React is not defined". Enable the automatic runtime in the bundler.
const enableAutomaticJsxRuntime = (webpackConfig: Configuration): void => {
  for (const rule of webpackConfig.module?.rules ?? []) {
    if (!rule || typeof rule !== 'object' || !('use' in rule)) {
      continue;
    }
    const uses = Array.isArray(rule.use) ? rule.use : [rule.use];
    for (const use of uses) {
      if (typeof use === 'object' && use !== null && use.loader === 'swc-loader') {
        const options = (use.options ??= {}) as Record<string, any>;
        const jsc = (options.jsc ??= {});
        const transform = (jsc.transform ??= {});
        transform.react = { ...(transform.react ?? {}), runtime: 'automatic' };
      }
    }
  }
};

const config = async (env: Env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);
  enableAutomaticJsxRuntime(baseConfig);

  return merge(baseConfig, {
    resolve: {
      alias: {
        '@grafana/i18n$': path.resolve(__dirname, 'node_modules/@grafana/i18n'),
        'react-i18next$': path.resolve(__dirname, 'node_modules/react-i18next'),
      },
    },
  });
};

export default config;

// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';
const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const baseConfig = require('./.config/jest.config');

module.exports = {
  // Jest configuration provided by @grafana/create-plugin
  ...baseConfig,
  // Inform Jest to only transform specific node_module packages.
  transformIgnorePatterns: [nodeModulesToTransform([...grafanaESModules, 'monaco-promql'])],
  // Override the swc/jest transform to use the automatic JSX runtime so .tsx
  // files don't need to `import React`. Matches the tsconfig "jsx": "react-jsx".
  transform: {
    '^.+\\.(t|j)sx?$': [
      '@swc/jest',
      {
        sourceMaps: 'inline',
        jsc: {
          parser: {
            syntax: 'typescript',
            tsx: true,
            decorators: false,
            dynamicImport: true,
          },
          transform: {
            react: {
              runtime: 'automatic',
            },
          },
        },
      },
    ],
  },
};

// Jest setup provided by Grafana scaffolding
import './.config/jest-setup';
import { TextEncoder, TextDecoder } from 'util';

if (typeof window !== 'undefined' && !window.grafanaBootData) {
  window.grafanaBootData = {
    settings: {
      featureToggles: {},
    },
    user: {
      locale: 'en-US',
    },
    navTree: [],
  };
}

global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;

// jsdom doesn't ship IntersectionObserver; some @grafana/ui Select internals
// (react-select dropdown viewport detection) need it.
if (typeof global.IntersectionObserver === 'undefined') {
  global.IntersectionObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return [];
    }
  };
}

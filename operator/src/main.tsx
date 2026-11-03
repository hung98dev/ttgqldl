import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from '@/app/App';
import { resolveDemoMode } from '@/app/demoMode';
import { createHttpClient } from '@/api/http';
import { createHealthApi } from '@/api/health';
import { demoData } from '@/api/demo';
import '@/styles/global.css';

/**
 * Production entry point.
 *
 * Demo mode is only enabled by an explicit build-time flag, and the flag is
 * documented as development-only. A production bundle defaults to real services
 * with no fabricated data.
 */
// resolveDemoMode is false in any production bundle, even if the flag leaked.
const demo = resolveDemoMode(import.meta.env);
const baseUrl = import.meta.env['VITE_OPERATOR_API_BASE_URL'] ?? '';
const healthPollMs = Number.parseInt(import.meta.env['VITE_OPERATOR_HEALTH_POLL_MS'] ?? '30000', 10);

const client = createHttpClient({ baseUrl, timeoutMs: 4000 });
const api = { health: createHealthApi(client) };

const container = document.getElementById('root');
if (container === null) {
  throw new Error('root element is missing');
}

createRoot(container).render(
  <StrictMode>
    <App
      api={api}
      demo={demo}
      demoFixtures={demo ? demoData() : undefined}
      apiBaseUrl={baseUrl}
      healthPollMs={Number.isFinite(healthPollMs) ? healthPollMs : 30_000}
    />
  </StrictMode>,
);

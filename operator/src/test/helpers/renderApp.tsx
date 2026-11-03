import { render, type RenderResult } from '@testing-library/react';
import { App } from '@/app/App';
import type { AuthAdapter } from '@/features/authentication/auth';
import type { DemoData } from '@/api/demo';
import type { HealthApi } from '@/api/health';
import type { Theme } from '@/app/providers/ThemeProvider';
import { okHealth, staticHealthApi } from '@/test/fixtures/health';

export interface RenderAppOptions {
  readonly route?: string;
  readonly demo?: boolean;
  readonly demoFixtures?: DemoData;
  readonly health?: HealthApi;
  readonly auth?: AuthAdapter;
  readonly theme?: Theme;
}

/** Renders the whole app with a MemoryRouter and deterministic dependencies. */
export function renderApp(options: RenderAppOptions = {}): RenderResult {
  return render(
    <App
      router="memory"
      initialEntries={[options.route ?? '/']}
      api={{ health: options.health ?? staticHealthApi(okHealth()) }}
      demo={options.demo ?? false}
      demoFixtures={options.demoFixtures}
      auth={options.auth}
      initialTheme={options.theme ?? 'light'}
      healthPollMs={0}
      apiBaseUrl=""
    />,
  );
}

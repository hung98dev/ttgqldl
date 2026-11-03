import { act, render, type RenderResult } from '@testing-library/react';
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

/**
 * Renders the whole app with a MemoryRouter and deterministic dependencies.
 *
 * The provider effects (health probe, session lookup) resolve asynchronously;
 * they are flushed inside act() before the result is returned so a test never
 * observes React state updates outside act.
 */
export async function renderApp(options: RenderAppOptions = {}): Promise<RenderResult> {
  const result = render(
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

  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });

  return result;
}

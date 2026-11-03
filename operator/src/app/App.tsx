import { BrowserRouter, MemoryRouter } from 'react-router-dom';
import { AppRoutes } from '@/app/routing/AppRoutes';
import { LocaleProvider } from '@/app/i18n/LocaleProvider';
import { ApiProvider, type OperatorApi } from '@/app/providers/ApiProvider';
import { AuthProvider } from '@/app/providers/AuthProvider';
import { HealthProvider } from '@/app/providers/HealthProvider';
import { ThemeProvider, type Theme } from '@/app/providers/ThemeProvider';
import { ErrorBoundary } from '@/components/feedback/ErrorBoundary';
import { demoData, type DemoData } from '@/api/demo';
import { unavailableAuthAdapter, type AuthAdapter } from '@/features/authentication/auth';

export interface AppProps {
  readonly api: OperatorApi;
  readonly demo?: boolean;
  readonly demoFixtures?: DemoData;
  readonly auth?: AuthAdapter;
  readonly apiBaseUrl?: string;
  readonly initialTheme?: Theme;
  readonly healthPollMs?: number;
  readonly router?: 'browser' | 'memory';
  readonly initialEntries?: readonly string[];
}

/**
 * Composition root.
 *
 * Everything stateful is injected: the API client, the demo flag, the auth
 * adapter, the router implementation. Production uses real HTTP and the
 * unavailable auth adapter; tests inject deterministic fakes.
 */
export function App({
  api,
  demo = false,
  demoFixtures,
  auth,
  apiBaseUrl = '',
  initialTheme,
  healthPollMs = 30_000,
  router = 'browser',
  initialEntries = ['/'],
}: AppProps) {
  const adapter = auth ?? unavailableAuthAdapter('M05 authentication is not implemented');

  const routes = <AppRoutes />;

  return (
    <ErrorBoundary>
      <LocaleProvider>
        <ThemeProvider initialTheme={initialTheme}>
          <ApiProvider
            api={api}
            demo={demo}
            demoData={demo ? (demoFixtures ?? demoData()) : null}
            apiBaseUrl={apiBaseUrl}
          >
            <AuthProvider adapter={adapter}>
              <HealthProvider pollMs={healthPollMs}>
                {router === 'memory' ? (
                  <MemoryRouter initialEntries={[...initialEntries]}>{routes}</MemoryRouter>
                ) : (
                  <BrowserRouter>{routes}</BrowserRouter>
                )}
              </HealthProvider>
            </AuthProvider>
          </ApiProvider>
        </ThemeProvider>
      </LocaleProvider>
    </ErrorBoundary>
  );
}

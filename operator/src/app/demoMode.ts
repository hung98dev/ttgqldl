/**
 * Demo-mode resolution and the build-time guard that keeps demonstration data
 * out of a production bundle.
 *
 * `VITE_OPERATOR_DEMO=true` is a development convenience. It is honoured only
 * when Vite is serving a development build; a production bundle resolves to
 * `false` even if the variable leaked into the environment, and `vite.config.ts`
 * refuses to build at all when the flag is set in production mode.
 */

export interface DemoEnvironment {
  readonly PROD?: boolean;
  readonly DEV?: boolean;
  readonly MODE?: string;
  readonly VITE_OPERATOR_DEMO?: string;
}

export function resolveDemoMode(env: DemoEnvironment): boolean {
  if (env.VITE_OPERATOR_DEMO !== 'true') {
    return false;
  }
  // Missing configuration is unsafe configuration: default to real services.
  if (env.PROD === true || env.MODE === 'production') {
    return false;
  }
  return true;
}

/**
 * Throws when a production build is asked to enable demonstration data.
 * Called from vite.config.ts so the failure happens before any bundle exists.
 */
export function assertDemoBuildAllowed(mode: string, flag: string | undefined): void {
  if (flag === 'true' && mode === 'production') {
    throw new Error(
      'VITE_OPERATOR_DEMO=true is forbidden for a production build: ' +
        'demonstration data must never ship. Build with the flag unset, or run the ' +
        'development server for the fixture preview.',
    );
  }
}

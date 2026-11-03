import { describe, expect, it } from 'vitest';
import { assertDemoBuildAllowed, resolveDemoMode } from '@/app/demoMode';

describe('demo mode resolution', () => {
  it('enables demo only in a development build with the explicit flag', () => {
    expect(resolveDemoMode({ DEV: true, MODE: 'development', VITE_OPERATOR_DEMO: 'true' })).toBe(true);
  });

  it('defaults to production behaviour when the flag is missing', () => {
    expect(resolveDemoMode({ DEV: true, MODE: 'development' })).toBe(false);
    expect(resolveDemoMode({})).toBe(false);
  });

  it('ignores a malformed flag value', () => {
    expect(resolveDemoMode({ DEV: true, VITE_OPERATOR_DEMO: 'yes' })).toBe(false);
    expect(resolveDemoMode({ DEV: true, VITE_OPERATOR_DEMO: '1' })).toBe(false);
    expect(resolveDemoMode({ DEV: true, VITE_OPERATOR_DEMO: 'TRUE' })).toBe(false);
  });

  it('never enables demo in a production bundle, even when the flag is set', () => {
    expect(resolveDemoMode({ PROD: true, MODE: 'production', VITE_OPERATOR_DEMO: 'true' })).toBe(false);
  });
});

describe('demo build guard', () => {
  it('fails a production build that asks for demo data', () => {
    expect(() => assertDemoBuildAllowed('production', 'true')).toThrow(/forbidden for a production build/);
  });

  it('allows production builds with the flag unset or false', () => {
    expect(() => assertDemoBuildAllowed('production', undefined)).not.toThrow();
    expect(() => assertDemoBuildAllowed('production', 'false')).not.toThrow();
  });

  it('allows the development server to enable the preview', () => {
    expect(() => assertDemoBuildAllowed('development', 'true')).not.toThrow();
  });
});

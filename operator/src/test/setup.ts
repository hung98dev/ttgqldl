import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach, beforeEach } from 'vitest';
import { currentViewport, setNarrowViewport } from './media';

/**
 * jsdom has no matchMedia and no layout engine. The console only asks one
 * question ("is the viewport narrow?"), so the mock answers exactly that and
 * tests switch it with setNarrowViewport.
 */
function installMatchMedia(): void {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: (query: string): MediaQueryList => {
      const matches = query.includes('max-width') ? currentViewport() === 'narrow' : false;
      return {
        matches,
        media: query,
        onchange: null,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        addListener: () => undefined,
        removeListener: () => undefined,
        dispatchEvent: () => false,
      } as MediaQueryList;
    },
  });

  window.scrollTo = () => undefined;
}

installMatchMedia();

beforeEach(() => {
  setNarrowViewport('wide');
});

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

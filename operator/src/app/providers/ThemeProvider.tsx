import { createContext, use, useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'ttgqldl.operator.theme';

interface ThemeContextValue {
  readonly theme: Theme;
  readonly toggle: () => void;
  readonly setTheme: (theme: Theme) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function readStoredTheme(): Theme | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw === 'light' || raw === 'dark' ? raw : null;
  } catch {
    // Storage can be unavailable (privacy mode); the default theme is fine.
    return null;
  }
}

function systemTheme(): Theme {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}

export function ThemeProvider({
  children,
  initialTheme,
}: {
  children: ReactNode;
  initialTheme?: Theme;
}) {
  const [theme, setThemeState] = useState<Theme>(() => initialTheme ?? readStoredTheme() ?? systemTheme());

  useEffect(() => {
    document.documentElement.dataset['theme'] = theme;
    try {
      window.localStorage.setItem(STORAGE_KEY, theme);
    } catch {
      // Persisting the theme is a convenience, never a requirement.
    }
  }, [theme]);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
  }, []);

  const toggle = useCallback(() => {
    setThemeState((current) => (current === 'dark' ? 'light' : 'dark'));
  }, []);

  const value = useMemo<ThemeContextValue>(() => ({ theme, toggle, setTheme }), [theme, toggle, setTheme]);

  return <ThemeContext value={value}>{children}</ThemeContext>;
}

export function useTheme(): ThemeContextValue {
  const context = use(ThemeContext);
  if (context === null) {
    throw new Error('useTheme must be used inside ThemeProvider');
  }
  return context;
}

export const THEME_STORAGE_KEY = STORAGE_KEY;

import { createContext, use, useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import type { AuthAdapter, AuthSession, Credentials } from '@/features/authentication/auth';

interface AuthContextValue {
  readonly session: AuthSession;
  readonly adapter: AuthAdapter;
  readonly pending: boolean;
  readonly login: (credentials: Credentials) => Promise<void>;
  readonly logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const INITIAL: AuthSession = { kind: 'unavailable', reason: 'M05 authentication is not implemented' };

export function AuthProvider({ children, adapter }: { children: ReactNode; adapter: AuthAdapter }) {
  const [session, setSession] = useState<AuthSession>(INITIAL);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    adapter
      .getSession(controller.signal)
      .then((next) => {
        if (!cancelled) {
          setSession(next);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setSession({ kind: 'unavailable', reason: 'session lookup failed' });
        }
      });
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [adapter]);

  const login = useCallback(
    async (credentials: Credentials) => {
      setPending(true);
      try {
        // A failed login must not fabricate a session: the adapter rejects and
        // the UI shows the real reason (M05 not implemented).
        const next = await adapter.login(credentials);
        setSession(next);
      } finally {
        setPending(false);
      }
    },
    [adapter],
  );

  const logout = useCallback(async () => {
    await adapter.logout();
    setSession({ kind: 'anonymous' });
  }, [adapter]);

  const value = useMemo<AuthContextValue>(
    () => ({ session, adapter, pending, login, logout }),
    [session, adapter, pending, login, logout],
  );

  return <AuthContext value={value}>{children}</AuthContext>;
}

export function useAuth(): AuthContextValue {
  const context = use(AuthContext);
  if (context === null) {
    throw new Error('useAuth must be used inside AuthProvider');
  }
  return context;
}

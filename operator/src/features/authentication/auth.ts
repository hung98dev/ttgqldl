import type { Role } from '@/contracts/api';
import { ApiError } from '@/api/errors';

/**
 * Authentication boundary.
 *
 * The real flow (password/token exchange, HttpOnly session cookie, CSRF header)
 * belongs to M05. Nothing here authenticates a user: the only adapter shipped in
 * M01 reports that the capability is unavailable, so no client-side check can be
 * mistaken for backend authorization.
 */
export type AuthSession =
  | { readonly kind: 'unavailable'; readonly reason: string }
  | { readonly kind: 'anonymous' }
  | { readonly kind: 'authenticated'; readonly role: Role; readonly displayName: string };

export interface Credentials {
  readonly account: string;
  readonly password: string;
}

export interface AuthAdapter {
  readonly id: string;
  getSession(signal?: AbortSignal): Promise<AuthSession>;
  login(credentials: Credentials, signal?: AbortSignal): Promise<AuthSession>;
  logout(): Promise<void>;
}

export function unavailableAuthAdapter(reason: string): AuthAdapter {
  return {
    id: 'unavailable',
    getSession: () => Promise.resolve({ kind: 'unavailable', reason }),
    login: () =>
      Promise.reject(
        new ApiError({
          kind: 'http',
          status: 501,
          code: 'AUTH_UNAVAILABLE',
          message: 'operator authentication is not implemented (M05)',
          retryable: false,
        }),
      ),
    logout: () => Promise.resolve(),
  };
}

/** UI-only permissions. The backend remains the authority for every action. */
export type Permission =
  | 'dashboard.view'
  | 'monitoring.view'
  | 'villages.view'
  | 'sessions.view'
  | 'moderation.view'
  | 'sessions.control'
  | 'moderation.mutate'
  | 'villages.create'
  | 'privacy.erase'
  | 'settings.manage';

const VIEWER_PERMISSIONS: readonly Permission[] = [
  'dashboard.view',
  'monitoring.view',
  'villages.view',
  'sessions.view',
  'moderation.view',
];

const OPERATOR_PERMISSIONS: readonly Permission[] = [...VIEWER_PERMISSIONS, 'sessions.control', 'moderation.mutate'];

const ADMIN_PERMISSIONS: readonly Permission[] = [
  ...OPERATOR_PERMISSIONS,
  'villages.create',
  'privacy.erase',
  'settings.manage',
];

export const ROLE_PERMISSIONS: Record<Role, readonly Permission[]> = {
  viewer: VIEWER_PERMISSIONS,
  operator: OPERATOR_PERMISSIONS,
  admin: ADMIN_PERMISSIONS,
};

export function effectiveRole(session: AuthSession): Role | null {
  return session.kind === 'authenticated' ? session.role : null;
}

export function hasPermission(session: AuthSession, permission: Permission): boolean {
  const role = effectiveRole(session);
  if (role === null) {
    return false;
  }
  return ROLE_PERMISSIONS[role].includes(permission);
}

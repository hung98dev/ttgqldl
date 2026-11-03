export { useAuth } from '@/app/providers/AuthProvider';
export {
  effectiveRole,
  hasPermission,
  ROLE_PERMISSIONS,
  unavailableAuthAdapter,
} from './auth';
export type { AuthAdapter, AuthSession, Credentials, Permission } from './auth';

import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';
import { ROLE_PERMISSIONS, hasPermission, unavailableAuthAdapter } from '@/features/authentication/auth';
import { fixedAuth } from '@/test/fixtures/health';

describe('role capabilities', () => {
  it('gives a viewer read-only permissions', async () => {
    expect(ROLE_PERMISSIONS.viewer).toContain('dashboard.view');
    expect(ROLE_PERMISSIONS.viewer).not.toContain('sessions.control');
    expect(ROLE_PERMISSIONS.viewer).not.toContain('villages.create');
    expect(ROLE_PERMISSIONS.viewer).not.toContain('privacy.erase');
  });

  it('gives an operator control but not administration', async () => {
    expect(ROLE_PERMISSIONS.operator).toContain('sessions.control');
    expect(ROLE_PERMISSIONS.operator).toContain('moderation.mutate');
    expect(ROLE_PERMISSIONS.operator).not.toContain('villages.create');
    expect(ROLE_PERMISSIONS.operator).not.toContain('settings.manage');
  });

  it('gives an admin every permission', async () => {
    expect(ROLE_PERMISSIONS.admin).toEqual(
      expect.arrayContaining([
        'villages.create',
        'privacy.erase',
        'settings.manage',
        'sessions.control',
        'dashboard.view',
      ]),
    );
  });

  it('denies every permission when no session exists', async () => {
    const session = { kind: 'unavailable', reason: 'M05' } as const;
    expect(hasPermission(session, 'dashboard.view')).toBe(false);
    expect(hasPermission(session, 'sessions.control')).toBe(false);
  });
});

describe('authentication boundary', () => {
  it('reports authentication as unavailable instead of inventing a session', async () => {
    const adapter = unavailableAuthAdapter('M05 authentication is not implemented');
    await expect(adapter.getSession()).resolves.toEqual({
      kind: 'unavailable',
      reason: 'M05 authentication is not implemented',
    });
    await expect(adapter.login({ account: 'a', password: 'b' })).rejects.toMatchObject({
      code: 'AUTH_UNAVAILABLE',
    });
  });

  it('shows the login form, rejects the attempt and stores no secret', async () => {
    const user = userEvent.setup();
    await renderApp({ route: '/login' });

    await user.type(await screen.findByLabelText('Tài khoản'), 'operator@example');
    await user.type(screen.getByLabelText('Mật khẩu'), 'super-secret');
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/operator authentication is not implemented/);

    const entries = Object.entries({ ...window.localStorage });
    expect(entries.every(([key]) => key === 'ttgqldl.operator.theme')).toBe(true);
    expect(JSON.stringify(entries)).not.toContain('super-secret');
    expect(JSON.stringify(entries)).not.toContain('operator@example');
  });

  it('shows the access-denied presentation', async () => {
    await renderApp({ route: '/access-denied' });
    expect(await screen.findByText('Không đủ quyền')).toBeInTheDocument();
  });

  it('reflects an authenticated role in the settings summary', async () => {
    await renderApp({ route: '/settings', auth: fixedAuth('admin') });
    expect(await screen.findByText(/Quản trị/)).toBeInTheDocument();
  });
});

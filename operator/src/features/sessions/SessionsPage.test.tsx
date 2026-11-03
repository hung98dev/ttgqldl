import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';
import { fixedAuth } from '@/test/fixtures/health';

describe('sessions', () => {
  it('shows the unavailable state and disabled controls in production', async () => {
    renderApp({ route: '/sessions', demo: false });

    expect(await screen.findByText(/Session listing depends on M05/)).toBeInTheDocument();
    expect(screen.getByText(/Điều khiển phiên cần API M05/)).toBeInTheDocument();
  });

  it('disables controls when no operator role is authenticated', async () => {
    renderApp({ route: '/sessions', demo: true });

    const stop = await screen.findByRole('button', { name: /Dừng hẳn demo-session-01/ });
    expect(stop).toBeDisabled();
  });

  it('allows an operator to pause and resume a demo session', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/sessions', demo: true, auth: fixedAuth('operator') });

    const pause = await screen.findByRole('button', { name: /Tạm dừng demo-session-01/ });
    expect(pause).toBeEnabled();
    await user.click(pause);

    // Pausing flips the available control, which proves the state changed
    // without relying on duplicated label text.
    expect(await screen.findByRole('button', { name: /Tiếp tục demo-session-01/ })).toBeEnabled();
    expect(screen.getByRole('button', { name: /Tạm dừng demo-session-01/ })).toBeDisabled();
    expect(screen.getByRole('status')).toHaveTextContent('Mô phỏng cục bộ');
  });

  it('asks for confirmation before stopping a session and can be cancelled', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/sessions', demo: true, auth: fixedAuth('operator') });

    await user.click(await screen.findByRole('button', { name: /Dừng hẳn demo-session-01/ }));
    const dialog = await screen.findByRole('dialog', { name: 'Dừng phiên chơi?' });
    expect(within(dialog).getByText(/không thể hoàn tác/)).toBeInTheDocument();

    await user.click(within(dialog).getByRole('button', { name: 'Huỷ' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.queryByText('Đã dừng')).not.toBeInTheDocument();
  });

  it('stops the session only after confirmation', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/sessions', demo: true, auth: fixedAuth('operator') });

    await user.click(await screen.findByRole('button', { name: /Dừng hẳn demo-session-01/ }));
    const dialog = await screen.findByRole('dialog', { name: 'Dừng phiên chơi?' });
    await user.click(within(dialog).getByRole('button', { name: 'Dừng hẳn' }));

    const table = screen.getByRole('table', { name: 'Phiên chơi' });
    expect(within(table).getByText('Đã dừng')).toBeInTheDocument();
  });

  it('closes the dialog on Escape', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/sessions', demo: true, auth: fixedAuth('operator') });

    await user.click(await screen.findByRole('button', { name: /Dừng hẳn demo-session-01/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('shows an empty state when there are no sessions', async () => {
    renderApp({
      route: '/sessions',
      demo: true,
      auth: fixedAuth('operator'),
      demoFixtures: { villages: [], sessions: [], moderation: [], eraseRequests: [] },
    });

    expect(await screen.findByText('Chưa có phiên chơi')).toBeInTheDocument();
  });
});

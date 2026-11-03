import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';
import { degradedHealth, failingHealthApi, okHealth } from '@/test/fixtures/health';
import { ApiError } from '@/api/errors';
import type { HealthApi } from '@/api/health';

function main(): HTMLElement {
  return screen.getByRole('main');
}

describe('dashboard', () => {
  it('shows backend and PostgreSQL readiness from the real endpoints', async () => {
    await renderApp();

    expect(await screen.findByText('Backend phản hồi')).toBeInTheDocument();
    expect(within(main()).getByText('PostgreSQL sẵn sàng')).toBeInTheDocument();
    expect(within(main()).getByText(/writer_epoch 42/)).toBeInTheDocument();
  });

  it('shows DEGRADED when readiness answers 503 and recovers on retry', async () => {
    const user = userEvent.setup();
    let calls = 0;
    const health: HealthApi = {
      check: () => {
        calls += 1;
        return Promise.resolve(calls === 1 ? degradedHealth() : okHealth());
      },
    };

    await renderApp({ health });

    expect((await screen.findAllByText('PostgreSQL chưa sẵn sàng')).length).toBeGreaterThan(0);
    expect(within(main()).getByText(/readiness failed: postgres/)).toBeInTheDocument();

    await user.click(within(main()).getByRole('button', { name: 'Kiểm tra lại' }));
    expect(await within(main()).findByText('PostgreSQL sẵn sàng')).toBeInTheDocument();
    expect(calls).toBe(2);
  });

  it('shows an error state with retry when the backend is unreachable', async () => {
    const user = userEvent.setup();
    let calls = 0;
    const health: HealthApi = {
      check: () => {
        calls += 1;
        return calls === 1
          ? Promise.reject(new ApiError({ kind: 'network', message: 'connection refused' }))
          : Promise.resolve(okHealth());
      },
    };

    await renderApp({ health });

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Không kết nối được tới backend');

    await user.click(within(alert).getByRole('button', { name: 'Kiểm tra lại' }));
    expect(await within(main()).findByText('PostgreSQL sẵn sàng')).toBeInTheDocument();
  });

  it('reports only the two implemented capabilities as available', async () => {
    await renderApp();

    const table = await screen.findByRole('table', { name: 'Năng lực hiện có' });
    const rows = within(table).getAllByRole('row').slice(1);
    const available = rows.filter((row) => within(row).queryByText('AVAILABLE') !== null);

    expect(available).toHaveLength(2);
    expect(within(available[0] as HTMLElement).getByText('health.read')).toBeInTheDocument();
    expect(within(available[1] as HTMLElement).getByText('readiness.read')).toBeInTheDocument();
  });

  it('never fabricates a session in production mode', async () => {
    await renderApp({ demo: false });

    expect(await screen.findByText('Phiên chơi thực tế')).toBeInTheDocument();
    expect(
      within(main()).getByText('Chưa có phiên chơi nào. Hệ thống chưa tạo dữ liệu trò chơi.'),
    ).toBeInTheDocument();
    expect(within(main()).queryByText('Đang chạy')).not.toBeInTheDocument();
  });

  it('lists the LIVE approvals gate', async () => {
    await renderApp();
    expect(await screen.findByText(/GATE-LIVE đang bị chặn/)).toBeInTheDocument();
  });

  it('keeps the timeout failure distinguishable from a network failure', async () => {
    await renderApp({ health: failingHealthApi(new ApiError({ kind: 'timeout', message: 'took too long' })) });

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Backend không phản hồi trong thời gian cho phép');
  });
});

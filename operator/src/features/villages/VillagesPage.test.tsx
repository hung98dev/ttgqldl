import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';
import { emptyDemoData } from '@/test/fixtures/health';

describe('villages', () => {
  it('shows the unavailable state and a disabled form in production', async () => {
    await renderApp({ route: '/villages', demo: false });

    expect((await screen.findAllByText(/admin\/v1\/villages/)).length).toBeGreaterThan(0);
    expect(screen.getByLabelText(/Tên làng/)).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Tạo làng' })).toBeDisabled();
    expect(screen.getByText(/Tạo làng cần endpoint POST/)).toBeInTheDocument();
  });

  it('lists fixture villages in demo mode', async () => {
    await renderApp({ route: '/villages', demo: true });

    expect(await screen.findByText('Làng Ven Sông')).toBeInTheDocument();
    expect(screen.getByText('Làng Đồi Gió')).toBeInTheDocument();
    expect(screen.getByText('DEMO / DỮ LIỆU MÔ PHỎNG')).toBeInTheDocument();
  });

  it('shows an empty state when the fixture has no villages', async () => {
    await renderApp({ route: '/villages', demo: true, demoFixtures: emptyDemoData() });

    expect(await screen.findByText('Chưa có làng nào')).toBeInTheDocument();
  });

  it('validates the documented display_name constraint before touching any state', async () => {
    const user = userEvent.setup();
    await renderApp({ route: '/villages', demo: true });

    await user.click(screen.getByRole('button', { name: 'Tạo làng' }));
    expect(await screen.findByText('Tên làng là bắt buộc.')).toBeInTheDocument();

    await user.type(screen.getByLabelText(/Tên làng/), 'x'.repeat(65));
    await user.click(screen.getByRole('button', { name: 'Tạo làng' }));
    expect(await screen.findByText('Tên làng tối đa 64 ký tự.')).toBeInTheDocument();
  });

  it('offers only mock and replay, and disables authorized_live', async () => {
    await renderApp({ route: '/villages', demo: true });

    const select = await screen.findByLabelText('Nguồn dữ liệu');
    const options = Array.from(select.querySelectorAll('option'));
    expect(options.map((option) => option.value)).toEqual(['mock', 'replay', 'authorized_live']);
    const live = options.find((option) => option.value === 'authorized_live');
    expect(live).toBeDisabled();
  });

  it('adds a clearly simulated row in demo mode', async () => {
    const user = userEvent.setup();
    await renderApp({ route: '/villages', demo: true });

    await user.type(await screen.findByLabelText(/Tên làng/), 'Làng Mới');
    await user.click(screen.getByRole('button', { name: 'Tạo làng' }));

    const table = await screen.findByRole('table', { name: 'Làng' });
    expect(within(table).getByText('Làng Mới')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('Mô phỏng cục bộ');
  });
});

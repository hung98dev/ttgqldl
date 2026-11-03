import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';
import { THEME_STORAGE_KEY } from '@/app/providers/ThemeProvider';

describe('app shell', () => {
  it('starts on the dashboard with navigation and backend status', async () => {
    renderApp();

    expect(await screen.findByRole('heading', { level: 2, name: 'Tổng quan' })).toBeInTheDocument();
    const nav = screen.getByRole('navigation', { name: 'Điều hướng chính' });
    expect(within(nav).getAllByRole('link')).toHaveLength(6);
    expect((await screen.findAllByText('PostgreSQL sẵn sàng')).length).toBeGreaterThan(0);
  });

  it('navigates between the six sections', async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole('link', { name: 'Làng' }));
    expect(await screen.findByRole('heading', { level: 2, name: 'Làng' })).toBeInTheDocument();

    await user.click(screen.getByRole('link', { name: 'Kiểm duyệt' }));
    expect(await screen.findByRole('heading', { level: 2, name: 'Kiểm duyệt' })).toBeInTheDocument();
  });

  it('marks the active section with aria-current', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/sessions' });

    const active = await screen.findByRole('link', { name: 'Phiên chơi' });
    await user.click(active);
    expect(active).toHaveAttribute('aria-current', 'page');
  });

  it('toggles the theme and persists only the theme preference', async () => {
    const user = userEvent.setup();
    renderApp({ theme: 'light' });

    expect(document.documentElement.dataset['theme']).toBe('light');
    await user.click(screen.getByRole('button', { name: 'Chuyển sang giao diện tối' }));
    expect(document.documentElement.dataset['theme']).toBe('dark');
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('offers a skip link that targets the main landmark', () => {
    renderApp();
    const skip = screen.getByRole('link', { name: 'Bỏ qua điều hướng, tới nội dung chính' });
    expect(skip).toHaveAttribute('href', '#main-content');
    const main = document.getElementById('main-content');
    expect(main?.tagName).toBe('MAIN');
    expect(main).toHaveAttribute('tabindex', '-1');
  });

  it('collapses the sidebar when the viewport is narrow', async () => {
    const { setNarrowViewport } = await import('@/test/media');
    setNarrowViewport('narrow');
    renderApp();

    const shell = document.querySelector('.shell');
    expect(shell).toHaveClass('shell--collapsed');
    expect(document.querySelector('.sidebar')).toHaveAttribute('data-collapsed', 'true');
  });

  it('reports an unreachable backend without inventing readiness', async () => {
    const { failingHealthApi } = await import('@/test/fixtures/health');
    const { ApiError } = await import('@/api/errors');
    renderApp({ health: failingHealthApi(new ApiError({ kind: 'network', message: 'refused' })) });

    expect(await screen.findByRole('alert')).toHaveTextContent('Không kết nối được tới backend');
  });
});

describe('demo mode labelling', () => {
  it('shows the demo banner and demo badge when enabled', async () => {
    renderApp({ demo: true });

    expect(await screen.findByText('DEMO / DỮ LIỆU MÔ PHỎNG')).toBeInTheDocument();
    expect(screen.getByText(/fixture cố định/)).toBeInTheDocument();
  });

  it('does not show the demo banner in the default production mode', async () => {
    renderApp({ demo: false });

    await screen.findByRole('heading', { level: 2, name: 'Tổng quan' });
    expect(screen.queryByText('DEMO / DỮ LIỆU MÔ PHỎNG')).not.toBeInTheDocument();
  });
});

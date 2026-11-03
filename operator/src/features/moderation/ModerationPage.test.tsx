import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/helpers/renderApp';

describe('moderation', () => {
  it('never renders an untrusted alias as markup', async () => {
    renderApp({ route: '/moderation', demo: true });

    const table = await screen.findByRole('table', { name: 'Kiểm duyệt' });
    const hostile = within(table).getByText('<img src=x onerror="alert(1)">');
    expect(hostile.tagName).toBe('SPAN');
    expect(document.querySelector('img')).toBeNull();
    expect(hostile.innerHTML).not.toContain('<img');
  });

  it('filters participants by alias without calling the backend', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/moderation', demo: true });

    const search = await screen.findByRole('textbox', { name: 'Tìm người tham gia' });
    await user.type(search, '003');

    const table = screen.getByRole('table', { name: 'Kiểm duyệt' });
    expect(within(table).getByText('Người xem 003')).toBeInTheDocument();
    expect(within(table).queryByText('Người xem 001')).not.toBeInTheDocument();
  });

  it('disables hide/block actions while the endpoint is unavailable', async () => {
    renderApp({ route: '/moderation', demo: true });

    const hides = await screen.findAllByRole('button', { name: 'Ẩn' });
    expect(hides.length).toBeGreaterThan(0);
    for (const button of hides) {
      expect(button).toBeDisabled();
      expect(button).toHaveAttribute('title', expect.stringContaining('M07'));
    }
  });

  it('shows the privacy erase queue with its real states', async () => {
    renderApp({ route: '/moderation', demo: true });

    expect(await screen.findByText('Đã áp dụng')).toBeInTheDocument();
    expect(screen.getByText('Bị chặn')).toBeInTheDocument();
    expect(screen.getByText(/Cần quyền admin/)).toBeInTheDocument();
  });

  it('shows an unavailable state in production instead of fake participants', async () => {
    renderApp({ route: '/moderation', demo: false });

    expect((await screen.findAllByText(/Participant search is M07 work/)).length).toBeGreaterThan(0);
    expect(screen.getByRole('textbox', { name: 'Tìm người tham gia' })).toBeDisabled();
    expect(screen.queryByText('Người xem 001')).not.toBeInTheDocument();
  });

  it('reports no results for a search that matches nothing', async () => {
    const user = userEvent.setup();
    renderApp({ route: '/moderation', demo: true });

    await user.type(await screen.findByRole('textbox', { name: 'Tìm người tham gia' }), 'zzz');
    expect(await screen.findByText('Không tìm thấy người tham gia phù hợp.')).toBeInTheDocument();
  });
});

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ErrorBoundary } from '@/components/feedback/ErrorBoundary';

function Exploding(): never {
  throw new Error('render blew up');
}

describe('error boundary', () => {
  it('renders a fallback instead of a blank page', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    render(
      <ErrorBoundary>
        <Exploding />
      </ErrorBoundary>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Giao diện gặp lỗi không mong đợi');
    expect(screen.getByText('render blew up')).toBeInTheDocument();
    spy.mockRestore();
  });

  it('offers a reload action', async () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const reload = vi.fn();

    render(
      <ErrorBoundary onReload={reload}>
        <Exploding />
      </ErrorBoundary>,
    );

    await userEvent.setup().click(screen.getByRole('button', { name: 'Tải lại trang' }));
    expect(reload).toHaveBeenCalledTimes(1);
    spy.mockRestore();
  });
});

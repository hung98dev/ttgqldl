import { useCallback, useEffect, useId, useRef, type ReactNode } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';

const FOCUSABLE = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

/**
 * Accessible confirmation dialog.
 *
 * Uses an explicit role="dialog" container instead of <dialog> so behaviour is
 * identical in browsers and in jsdom, and implements focus capture, Tab
 * cycling, Escape to cancel and focus restoration.
 */
export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel,
  tone = 'danger',
  busy = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  description?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: 'danger' | 'primary';
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const t = useT();
  const titleId = useId();
  const descriptionId = useId();
  const containerRef = useRef<HTMLDivElement | null>(null);

  const focusables = useCallback((): HTMLElement[] => {
    const root = containerRef.current;
    if (root === null) {
      return [];
    }
    return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE));
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }

    const previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const first = focusables()[0];
    first?.focus();

    const onKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onCancel();
        return;
      }
      if (event.key !== 'Tab') {
        return;
      }
      const items = focusables();
      if (items.length === 0) {
        return;
      }
      const firstItem = items[0];
      const lastItem = items[items.length - 1];
      const active = document.activeElement;
      if (event.shiftKey && active === firstItem) {
        event.preventDefault();
        lastItem?.focus();
      } else if (!event.shiftKey && active === lastItem) {
        event.preventDefault();
        firstItem?.focus();
      }
    };

    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      previouslyFocused?.focus();
    };
  }, [open, focusables, onCancel]);

  if (!open) {
    return null;
  }

  return (
    <div className="dialog-backdrop">
      <div
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        ref={containerRef}
      >
        <h2 className="dialog__title" id={titleId}>
          {title}
        </h2>
        {description ? (
          <div className="dialog__body" id={descriptionId}>
            {description}
          </div>
        ) : null}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onCancel} disabled={busy}>
            {cancelLabel ?? t('action.cancel')}
          </button>
          <button
            type="button"
            className={tone === 'danger' ? 'btn btn--danger' : 'btn btn--primary'}
            onClick={onConfirm}
            disabled={busy}
          >
            {confirmLabel ?? t('action.confirm')}
          </button>
        </div>
      </div>
    </div>
  );
}

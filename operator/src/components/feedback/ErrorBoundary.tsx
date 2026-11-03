import { Component, type ErrorInfo, type ReactNode } from 'react';
import { vi } from '@/app/i18n/messages';

interface Props {
  readonly children: ReactNode;
  /** Test/embedding seam; defaults to a full page reload. */
  readonly onReload?: () => void;
}

interface State {
  readonly error: Error | null;
}

/**
 * Last-resort boundary: a rendering failure must not blank the console or look
 * like a healthy page. It never reports the failure to a backend and never
 * swallows the error silently.
 */
export class ErrorBoundary extends Component<Props, State> {
  override state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('[ttgqldl.operator] render failure', error.message, info.componentStack);
  }

  override render(): ReactNode {
    if (this.state.error === null) {
      return this.props.children;
    }

    return (
      <div className="error-boundary" role="alert">
        <h1>{vi['error.boundary.title']}</h1>
        <p>{vi['error.boundary.body']}</p>
        <pre className="error-boundary__detail">{this.state.error.message}</pre>
        <button
          type="button"
          className="btn btn--primary"
          onClick={() => {
            if (this.props.onReload) {
              this.props.onReload();
            } else {
              window.location.reload();
            }
          }}
        >
          {vi['error.boundary.reload']}
        </button>
      </div>
    );
  }
}

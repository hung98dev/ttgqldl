import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  readonly variant?: Variant;
  readonly busy?: boolean;
  readonly icon?: ReactNode;
}

export function Button({
  variant = 'secondary',
  busy = false,
  icon,
  children,
  className,
  disabled,
  ...rest
}: ButtonProps) {
  const classes = ['btn', `btn--${variant}`, className].filter(Boolean).join(' ');
  return (
    <button
      type="button"
      className={classes}
      disabled={disabled ?? busy}
      aria-busy={busy || undefined}
      {...rest}
    >
      {icon ? (
        <span className="btn__icon" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <span>{children}</span>
    </button>
  );
}

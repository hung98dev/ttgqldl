import { useState, type FormEvent } from 'react';
import { useT } from '@/app/i18n/LocaleProvider';
import { isApiError } from '@/api/errors';
import { Button } from '@/components/common/Button';
import { Card } from '@/components/common/Card';
import { TextField } from '@/components/common/TextField';
import { useAuth } from '@/features/authentication/useAuth';

/**
 * Login presentation only.
 *
 * The form is wired to the injected AuthAdapter, which in M01 always rejects
 * with AUTH_UNAVAILABLE. Nothing is stored: no token, no password, no session
 * id is written to localStorage, sessionStorage or a cookie by this page.
 */
export function LoginPage() {
  const t = useT();
  const { login, pending, session } = useAuth();
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [message, setMessage] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setMessage(null);
    try {
      await login({ account, password });
      setPassword('');
    } catch (cause) {
      setPassword('');
      setMessage(isApiError(cause) ? cause.message : t('auth.login.unavailable'));
    }
  }

  return (
    <div className="page page--narrow">
      <header className="page__header">
        <h2 className="page__title">{t('auth.login.title')}</h2>
        <p className="page__subtitle">{t('auth.login.subtitle')}</p>
      </header>

      <Card>
        <p className="notice notice--warning" role="note">
          {t('auth.login.unavailable')}
        </p>

        <form className="form" onSubmit={(event) => void onSubmit(event)} noValidate>
          <TextField
            label={t('auth.login.email')}
            value={account}
            onChange={setAccount}
            autoComplete="username"
          />
          <TextField
            label={t('auth.login.password')}
            value={password}
            onChange={setPassword}
            type="password"
            autoComplete="current-password"
          />
          <div className="form__actions">
            <Button type="submit" variant="primary" busy={pending}>
              {t('auth.login.submit')}
            </Button>
          </div>
        </form>

        {message ? (
          <p className="notice notice--danger" role="alert">
            {message}
          </p>
        ) : null}

        <p className="muted">
          {session.kind === 'unavailable' ? session.reason : t('auth.role.unknown')}
        </p>
      </Card>
    </div>
  );
}

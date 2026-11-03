import { createContext, use, type ReactNode } from 'react';
import { vi, type LocaleMessages, type MessageKey } from './messages';

export type Translate = (key: MessageKey, params?: Record<string, string | number>) => string;

interface LocaleContextValue {
  readonly locale: string;
  readonly t: Translate;
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined ? match : String(value);
  });
}

export function createTranslator(messages: LocaleMessages): Translate {
  return (key, params) => interpolate(messages[key], params);
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const value: LocaleContextValue = { locale: 'vi', t: createTranslator(vi) };
  return <LocaleContext value={value}>{children}</LocaleContext>;
}

export function useT(): Translate {
  const context = use(LocaleContext);
  if (context === null) {
    throw new Error('useT must be used inside LocaleProvider');
  }
  return context.t;
}

export function useLocale(): string {
  const context = use(LocaleContext);
  if (context === null) {
    throw new Error('useLocale must be used inside LocaleProvider');
  }
  return context.locale;
}

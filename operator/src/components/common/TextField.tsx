import { useId, type ChangeEvent, type HTMLInputTypeAttribute } from 'react';

export interface TextFieldProps {
  readonly label: string;
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly error?: string;
  readonly hint?: string;
  readonly type?: HTMLInputTypeAttribute;
  readonly placeholder?: string;
  readonly disabled?: boolean;
  readonly required?: boolean;
  readonly autoComplete?: string;
  readonly inputMode?: 'text' | 'numeric' | 'decimal';
  readonly name?: string;
}

export function TextField({
  label,
  value,
  onChange,
  error,
  hint,
  type = 'text',
  placeholder,
  disabled = false,
  required = false,
  autoComplete,
  inputMode,
  name,
}: TextFieldProps) {
  const id = useId();
  const errorId = `${id}-error`;
  const hintId = `${id}-hint`;
  const describedBy = [hint ? hintId : null, error ? errorId : null].filter(Boolean).join(' ');

  const handleChange = (event: ChangeEvent<HTMLInputElement>): void => {
    onChange(event.target.value);
  };

  return (
    <div className={`field ${error ? 'field--invalid' : ''}`}>
      <label className="field__label" htmlFor={id}>
        {label}
        {required ? <span aria-hidden="true"> *</span> : null}
      </label>
      <input
        id={id}
        name={name}
        className="field__input"
        type={type}
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        disabled={disabled}
        required={required}
        autoComplete={autoComplete}
        inputMode={inputMode}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy === '' ? undefined : describedBy}
      />
      {hint ? (
        <p className="field__hint" id={hintId}>
          {hint}
        </p>
      ) : null}
      {error ? (
        <p className="field__error" id={errorId} role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}

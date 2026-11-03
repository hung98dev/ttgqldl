import { describe, expect, it } from 'vitest';
import { parseReadyResponse } from '@/api/guards';
import { ApiError } from '@/api/errors';

describe('readiness protocol validation', () => {
  it('accepts a safe non-negative integer writer_epoch', () => {
    expect(parseReadyResponse({ status: 'ready', writer_epoch: 0 })).toEqual({
      status: 'ready',
      writer_epoch: 0,
    });
    expect(parseReadyResponse({ status: 'ready', writer_epoch: 9007199254740991 })).toEqual({
      status: 'ready',
      writer_epoch: 9007199254740991,
    });
  });

  it('rejects a floating point writer_epoch', () => {
    expect(() => parseReadyResponse({ status: 'ready', writer_epoch: 1.5 })).toThrow(ApiError);
    expect(() => parseReadyResponse({ status: 'ready', writer_epoch: 1.5 })).toThrow(/must be an integer/);
  });

  it('rejects a negative writer_epoch', () => {
    expect(() => parseReadyResponse({ status: 'ready', writer_epoch: -1 })).toThrow(/must not be negative/);
  });

  it('rejects a writer_epoch beyond the safe integer range', () => {
    expect(() => parseReadyResponse({ status: 'ready', writer_epoch: 9007199254740992 })).toThrow(
      /safe integer/,
    );
  });

  it('rejects a string writer_epoch and a missing field', () => {
    expect(() => parseReadyResponse({ status: 'ready', writer_epoch: '42' })).toThrow(/finite number/);
    expect(() => parseReadyResponse({ status: 'ready' })).toThrow(/finite number/);
  });

  it('rejects an unexpected status value', () => {
    expect(() => parseReadyResponse({ status: 'alive', writer_epoch: 1 })).toThrow(/unexpected readiness.status/);
  });
});

import type { ApiErrorEnvelope } from '@/contracts/api';
import type { HealthResponse, ReadyResponse } from '@/contracts/health';
import { ApiError } from './errors';

/**
 * Hand-written validators for externally sourced JSON.
 *
 * No `any` and no blind casts: every field is checked and a shape violation is
 * a protocol error, so a malformed backend response can never be rendered as
 * plausible data.
 */

function asRecord(value: unknown, what: string): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw ApiError.protocol(`${what} must be a JSON object`);
  }
  return value as Record<string, unknown>;
}

function asString(value: unknown, what: string): string {
  if (typeof value !== 'string') {
    throw ApiError.protocol(`${what} must be a string`);
  }
  return value;
}

function asFiniteNumber(value: unknown, what: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw ApiError.protocol(`${what} must be a finite number`);
  }
  return value;
}

function asSafeInteger(value: unknown, what: string): number {
  const numeric = asFiniteNumber(value, what);
  if (!Number.isInteger(numeric)) {
    throw ApiError.protocol(`${what} must be an integer`);
  }
  if (numeric < 0) {
    throw ApiError.protocol(`${what} must not be negative`);
  }
  if (numeric > Number.MAX_SAFE_INTEGER) {
    throw ApiError.protocol(`${what} must be a safe integer`);
  }
  return numeric;
}

function asBoolean(value: unknown, what: string): boolean {
  if (typeof value !== 'boolean') {
    throw ApiError.protocol(`${what} must be a boolean`);
  }
  return value;
}

export function parseHealthResponse(json: unknown): HealthResponse {
  const body = asRecord(json, 'health response');
  const status = asString(body['status'], 'health.status');
  if (status !== 'alive') {
    throw ApiError.protocol(`unexpected health.status ${JSON.stringify(status)}`);
  }
  return { status: 'alive' };
}

export function parseReadyResponse(json: unknown): ReadyResponse {
  const body = asRecord(json, 'readiness response');
  const status = asString(body['status'], 'readiness.status');
  if (status !== 'ready') {
    throw ApiError.protocol(`unexpected readiness.status ${JSON.stringify(status)}`);
  }
  return {
    status: 'ready',
    writer_epoch: asSafeInteger(body['writer_epoch'], 'readiness.writer_epoch'),
  };
}

/** Returns the envelope when the payload matches, otherwise null. */
export function parseErrorEnvelope(json: unknown): ApiErrorEnvelope | null {
  if (typeof json !== 'object' || json === null) {
    return null;
  }
  const body = json as Record<string, unknown>;
  const error = body['error'];
  if (typeof error !== 'object' || error === null) {
    return null;
  }
  const detail = error as Record<string, unknown>;
  if (typeof detail['code'] !== 'string' || typeof detail['message'] !== 'string') {
    return null;
  }
  return {
    error: {
      code: detail['code'],
      message: detail['message'],
      request_id: typeof detail['request_id'] === 'string' ? detail['request_id'] : '',
      retryable: detail['retryable'] === true,
    },
  };
}

export { asBoolean, asFiniteNumber, asRecord, asSafeInteger, asString };

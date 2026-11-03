/**
 * Shared invariant helpers for tests. They mirror the backend rules that the
 * operator UI must never contradict.
 */
export const PublicStateRules = {
  /** wave_no is 1-based on the wire. */
  waveNumber(value: number): number {
    if (!Number.isInteger(value) || value < 1) {
      throw new Error(`wave_no must be an integer >= 1, got ${String(value)}`);
    }
    return value;
  },
} as const;

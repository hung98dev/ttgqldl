import { describe, expect, it } from 'vitest';
import { demoData } from '@/api/demo';
import { PublicStateRules } from '@/test/fixtures/rules';

/**
 * Demo fixtures are data too: they must satisfy the invariants the backend
 * enforces, otherwise the preview teaches the wrong model of the game.
 */
describe('demo fixtures', () => {
  const data = demoData();

  it('never publishes a wave number below one', () => {
    for (const village of data.villages) {
      expect(PublicStateRules.waveNumber(village.wave_no)).toBeGreaterThanOrEqual(1);
    }
    for (const session of data.sessions) {
      expect(PublicStateRules.waveNumber(session.wave_no)).toBeGreaterThanOrEqual(1);
    }
  });

  it('keeps every fixture clearly labelled as mock or replay', () => {
    for (const village of data.villages) {
      expect(['mock', 'replay']).toContain(village.source_status);
    }
    for (const session of data.sessions) {
      expect(['mock', 'replay']).toContain(session.source_status);
    }
  });

  it('uses opaque public references, never actor keys or provider ids', () => {
    for (const target of data.moderation) {
      expect(target.public_ref).not.toMatch(/^[0-9a-f]{64}$/i);
      expect(target.public_ref).toMatch(/^demo-ref-/);
    }
  });

  it('has deterministic values across calls', () => {
    expect(JSON.stringify(demoData())).toBe(JSON.stringify(demoData()));
  });
});

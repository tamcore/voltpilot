import { describe, it, expect } from 'vitest';
import { normalizeDeg, circularLowPass, androidHeadingFromAlpha } from './compass';

describe('compass math', () => {
	it('normalizes degrees into [0,360)', () => {
		expect(normalizeDeg(370)).toBe(10);
		expect(normalizeDeg(-10)).toBe(350);
		expect(normalizeDeg(0)).toBe(0);
	});

	it('low-pass takes the first sample verbatim', () => {
		expect(circularLowPass(null, 123)).toBeCloseTo(123);
	});

	it('low-pass moves toward the target along the shortest arc', () => {
		// from 350 toward 10 should increase past 360 → small positive angle
		const h = circularLowPass(350, 10, 0.5);
		expect(h).toBeCloseTo(0, 0);
	});

	it('low-pass does not jump the long way around', () => {
		const h = circularLowPass(10, 350, 0.5);
		// shortest arc is backwards to 0 then 350 → ~0
		expect(h).toBeCloseTo(0, 0);
	});

	it('android alpha → clockwise-from-north heading', () => {
		expect(androidHeadingFromAlpha(0)).toBe(0); // facing north
		expect(androidHeadingFromAlpha(90)).toBe(270); // alpha CCW 90 → heading 270
		expect(androidHeadingFromAlpha(90, 90)).toBe(0); // landscape correction
	});
});

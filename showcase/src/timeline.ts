import { speeds, type Speed } from "./zooms";

export type Segment = Speed & { outFrom: number; outTo: number };

/**
 * @name segments
 * @description Resolves the speed map against the recording length, giving
 * every segment its start and end on the output timeline in seconds.
 * @example segments(25.2)[1].outFrom
 */
export function segments(duration: number): Segment[] {
	let cursor = 0;
	return speeds.map((speed) => {
		const to = Math.min(speed.to, duration);
		const outFrom = cursor;
		cursor += (to - speed.from) / speed.rate;
		return { ...speed, to, outFrom, outTo: cursor };
	});
}

/**
 * @name sourceTime
 * @description Maps a moment on the output timeline back to the recording.
 * @example sourceTime(segments(25.2), 13)
 */
export function sourceTime(list: Segment[], out: number) {
	const segment = list.find((s) => out < s.outTo) ?? list[list.length - 1];
	return segment.from + (out - segment.outFrom) * segment.rate;
}

export type Zoom = {
	from: number;
	to: number;
	scale: number;
	x: number;
	y: number;
};

export type Speed = {
	from: number;
	to: number;
	rate: number;
};

export const zooms: Zoom[] = [
	{ from: 8.9, to: 10, scale: 2, x: 0.2, y: 0.9 },
	{ from: 10.4, to: 11.5, scale: 1.35, x: 0.37, y: 0.35 },
];

export const speeds: Speed[] = [
	{ from: 0, to: 11.5, rate: 1 },
	{ from: 11.5, to: 17.2, rate: 3 },
	{ from: 17.2, to: Infinity, rate: 1 },
];

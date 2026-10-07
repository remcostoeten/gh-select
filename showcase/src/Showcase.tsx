import {
	AbsoluteFill,
	OffthreadVideo,
	Sequence,
	spring,
	staticFile,
	useCurrentFrame,
	useVideoConfig,
} from "remotion";
import { segments, sourceTime } from "./timeline";
import { zooms, type Zoom } from "./zooms";

const ease = { damping: 20, stiffness: 220, mass: 0.7 };

type Props = { duration: number };

function clamp(value: number, min: number, max: number) {
	return Math.min(max, Math.max(min, value));
}

function weight(zoom: Zoom, time: number, fps: number) {
	const enter = spring({ frame: (time - zoom.from) * fps, fps, config: ease });
	const leave = spring({ frame: (time - zoom.to) * fps, fps, config: ease });
	return Math.max(0, enter - leave);
}

export function Showcase({ duration }: Props) {
	const frame = useCurrentFrame();
	const { fps, width, height } = useVideoConfig();
	const list = segments(duration);
	const time = sourceTime(list, frame / fps);

	let scale = 1;
	let x = 0.5;
	let y = 0.5;
	let total = 0;
	for (const zoom of zooms) {
		const w = weight(zoom, time, fps);
		scale += (zoom.scale - 1) * w;
		x += (zoom.x - 0.5) * w;
		y += (zoom.y - 0.5) * w;
		total += w;
	}
	if (total > 1) {
		x = 0.5 + (x - 0.5) / total;
		y = 0.5 + (y - 0.5) / total;
	}

	const left = clamp(x * width - width / scale / 2, 0, width - width / scale);
	const top = clamp(y * height - height / scale / 2, 0, height - height / scale);

	return (
		<AbsoluteFill style={{ background: "#1e1e2e" }}>
			<AbsoluteFill
				style={{
					transform: `scale(${scale}) translate(${-left}px, ${-top}px)`,
					transformOrigin: "0 0",
				}}
			>
				{list.map((segment) => (
					<Sequence
						key={segment.from}
						from={Math.round(segment.outFrom * fps)}
						durationInFrames={Math.round((segment.outTo - segment.outFrom) * fps)}
					>
						<OffthreadVideo
							src={staticFile("raw.mp4")}
							startFrom={Math.round(segment.from * fps)}
							playbackRate={segment.rate}
						/>
					</Sequence>
				))}
			</AbsoluteFill>
		</AbsoluteFill>
	);
}

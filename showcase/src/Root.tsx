import { getVideoMetadata } from "@remotion/media-utils";
import { Composition, staticFile } from "remotion";
import { Showcase } from "./Showcase";
import { segments } from "./timeline";

const fps = 30;

export function Root() {
	return (
		<Composition
			id="Showcase"
			component={Showcase}
			width={1200}
			height={700}
			fps={fps}
			durationInFrames={1}
			defaultProps={{ duration: 0 }}
			calculateMetadata={async () => {
				const meta = await getVideoMetadata(staticFile("raw.mp4"));
				const list = segments(meta.durationInSeconds);
				return {
					durationInFrames: Math.ceil(list[list.length - 1].outTo * fps),
					props: { duration: meta.durationInSeconds },
				};
			}}
		/>
	);
}

import Image from "next/image";
import { inlineSvg } from "@/shared/config/inline-svg";
import { site } from "@/shared/config/site";
import type { LogoSource } from "@/shared/config/site";

type Mode = "only" | "light" | "dark";

type Props = {
  source: LogoSource;
  mode: Mode;
  alt: string;
  width: number;
  height: number;
};

const imageVisibility: Record<Mode, string | undefined> = {
  only: undefined,
  light: "dark:hidden",
  dark: "hidden dark:block",
};

const inlineVisibility: Record<Mode, string> = {
  only: "block shrink-0",
  light: "block shrink-0 dark:hidden",
  dark: "hidden shrink-0 dark:block",
};

function LogoImage({ source, mode, alt, width, height }: Props) {
  const labelProps = alt ? { role: "img", "aria-label": alt } : { "aria-hidden": true };

  if (typeof source !== "string") {
    const Icon = source;
    return <Icon width={width} height={height} className={imageVisibility[mode]} {...labelProps} />;
  }

  if (inlineSvg.test(source)) {
    return (
      <span
        className={`${inlineVisibility[mode]} [&>svg]:size-full`}
        style={{ width, height }}
        dangerouslySetInnerHTML={{ __html: source }}
        {...labelProps}
      />
    );
  }

  return (
    <Image
      src={source}
      alt={alt}
      width={width}
      height={height}
      unoptimized
      className={imageVisibility[mode]}
    />
  );
}

export function Logo() {
  const { logo } = site;
  if (!logo) return null;
  const alt = logo.hideName ? (logo.alt ?? site.name) : "";
  const size = { alt, width: logo.width, height: logo.height };

  if (!logo.darkSrc) return <LogoImage source={logo.src} mode="only" {...size} />;

  return (
    <>
      <LogoImage source={logo.src} mode="light" {...size} />
      <LogoImage source={logo.darkSrc} mode="dark" {...size} />
    </>
  );
}

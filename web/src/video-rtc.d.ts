import "react";

declare module "react" {
  namespace JSX {
    interface IntrinsicElements {
      "video-rtc": React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          src?: string;
          mode?: string;
          visibilityThreshold?: string;
          background?: string;
        },
        HTMLElement
      >;
    }
  }
}

// Asset imports resolve differently per bundler: electron-vite hands back a
// URL string, while Next.js returns a StaticImageData object (`.src`).
interface StaticImageAsset {
  src: string;
  height?: number;
  width?: number;
  blurDataURL?: string;
}

declare module "*.png" {
  const src: string | StaticImageAsset;
  export default src;
}

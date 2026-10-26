import brandMarkAsset from "../../assets/chimii-brand.png";

// Next.js returns StaticImageData; Vite returns the URL string directly. The
// explicit annotation keeps both consumers narrowing correctly.
const asset = brandMarkAsset as string | { src: string };

export const CHIMII_BRAND_MARK_SRC =
  typeof asset === "string" ? asset : asset.src;

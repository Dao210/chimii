import { readdir, readFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { LDRAW_CATALOG, LDRAW_CATALOG_VERSION } from "./catalog.generated";
import {
  LDRAW_THUMBNAIL_CATALOG_VERSION,
  LDRAW_THUMBNAIL_COLORS,
  LDRAW_THUMBNAIL_HEIGHT,
  LDRAW_THUMBNAIL_RENDER_PRESET_VERSION,
  LDRAW_THUMBNAILS,
  LDRAW_THUMBNAIL_WIDTH,
} from "./catalog-thumbnails.generated";
import { LDRAW_RENDER_PRESET_VERSION } from "../components/ldraw-render-preset";

const thumbnailDirectory = resolve(process.cwd(), "build/catalog/thumbnails");

describe("generated LDraw thumbnails", () => {
  it("covers every catalog part in every supported colour", () => {
    const expectedKeys = Object.keys(LDRAW_CATALOG).flatMap((ldrawID) => (
      LDRAW_THUMBNAIL_COLORS.map((colorCode) => `${ldrawID}:${colorCode}`)
    ));

    expect(Object.keys(LDRAW_THUMBNAILS).sort()).toEqual(expectedKeys.sort());
    expect(LDRAW_THUMBNAIL_CATALOG_VERSION).toBe(LDRAW_CATALOG_VERSION);
    expect(LDRAW_THUMBNAIL_RENDER_PRESET_VERSION).toBe(LDRAW_RENDER_PRESET_VERSION);
    expect(LDRAW_THUMBNAIL_WIDTH).toBe(640);
    expect(LDRAW_THUMBNAIL_HEIGHT).toBe(400);

    for (const [key, thumbnail] of Object.entries(LDRAW_THUMBNAILS)) {
      expect(thumbnail.src, key).toBeTruthy();
      expect(thumbnail.assetHash, key).toBe(LDRAW_CATALOG[thumbnail.ldrawID]?.hash);
    }
  });

  it("stores 60 non-empty WebP files", async () => {
    const files = (await readdir(thumbnailDirectory)).filter((file) => file.endsWith(".webp"));
    expect(files).toHaveLength(Object.keys(LDRAW_THUMBNAILS).length);

    for (const filename of files) {
      const image = await readFile(join(thumbnailDirectory, filename));
      expect(image.byteLength, filename).toBeGreaterThan(1024);
      expect(image.subarray(0, 4).toString("ascii"), filename).toBe("RIFF");
      expect(image.subarray(8, 12).toString("ascii"), filename).toBe("WEBP");
    }
  });
});

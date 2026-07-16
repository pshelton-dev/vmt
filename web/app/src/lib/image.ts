// Client-side photo downscaling: phone cameras produce 4-12 MB images that
// are wasted on a garage tracker. Resize to a sane bound before upload —
// saves NAS storage and mobile data. Non-images (and small images) pass
// through untouched; any failure falls back to the original file.

const MAX_DIM = 1600;
const QUALITY = 0.82;
const SKIP_BELOW = 500 * 1024; // small files aren't worth recompressing

export async function downscaleImage(file: File): Promise<File> {
  if (!file.type.startsWith("image/") || file.type === "image/gif") return file;
  if (file.size < SKIP_BELOW) return file;
  try {
    const bmp = await createImageBitmap(file);
    const scale = Math.min(1, MAX_DIM / Math.max(bmp.width, bmp.height));
    const w = Math.round(bmp.width * scale);
    const h = Math.round(bmp.height * scale);
    const canvas = document.createElement("canvas");
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext("2d");
    if (!ctx) return file;
    ctx.drawImage(bmp, 0, 0, w, h);
    bmp.close();
    const blob = await new Promise<Blob | null>((res) =>
      canvas.toBlob(res, "image/jpeg", QUALITY),
    );
    if (!blob || blob.size >= file.size) return file; // no win — keep original
    const name = file.name.replace(/\.[^.]+$/, "") + ".jpg";
    return new File([blob], name, { type: "image/jpeg" });
  } catch {
    return file;
  }
}

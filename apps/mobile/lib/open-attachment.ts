import { Linking } from "react-native";
import * as FileSystem from "expo-file-system/legacy";
import * as Sharing from "expo-sharing";
import { api } from "@/data/api";
import { resolveAttachmentUrl } from "./attachment-url";

/** Native browsers do not share SecureStore credentials. Download private
 * files into app cache, then let the system picker open/save the local copy. */
export async function openAttachment(rawUrl: string, filename = "attachment") {
  const uri = resolveAttachmentUrl(rawUrl);
  if (!uri || !/^https?:\/\//i.test(uri)) throw new Error("Attachment URL is unavailable");
  const headers = api.attachmentHeaders(uri);
  if (!headers.Authorization) { await Linking.openURL(uri); return; }
  if (!FileSystem.cacheDirectory || !(await Sharing.isAvailableAsync())) throw new Error("File sharing is unavailable");
  const root = `${FileSystem.cacheDirectory}attachments/`;
  await FileSystem.makeDirectoryAsync(root, { intermediates: true });
  // Android may finish the chooser before the receiving app reads the URI.
  // Retain shared files in private cache; prune old copies on the next download.
  const expiredBefore = Date.now() - 24 * 60 * 60 * 1000;
  for (const entry of await FileSystem.readDirectoryAsync(root)) {
    const createdAt = Number(entry.split("-")[0]);
    if (createdAt > 0 && createdAt < expiredBefore) {
      await FileSystem.deleteAsync(`${root}${entry}`, { idempotent: true });
    }
  }
  const directory = `${root}${Date.now()}-${Math.random().toString(36).slice(2)}/`;
  await FileSystem.makeDirectoryAsync(directory, { intermediates: true });
  let shared = false;
  try {
    const name = filename.replace(/[^a-zA-Z0-9._-]/g, "_").slice(0, 120) || "attachment";
    const file = await FileSystem.downloadAsync(uri, `${directory}${name}`, { headers });
    if (file.status !== 200) throw new Error(`Attachment download failed (${file.status})`);
    await Sharing.shareAsync(file.uri);
    shared = true;
  } finally {
    if (!shared) await FileSystem.deleteAsync(directory, { idempotent: true });
  }
}

import { readdir, readFile, stat, unlink, writeFile } from "node:fs/promises"
import path from "node:path"
import { fileURLToPath } from "node:url"

import sharp from "sharp"

const rootDir = path.resolve(fileURLToPath(new URL("..", import.meta.url)))
const distDir = path.join(rootDir, "dist")

const imageExtensions = new Set([".png", ".jpg", ".jpeg"])
const textExtensions = new Set([".html", ".css", ".js", ".mjs", ".json"])
const maxImageWidth = Number(process.env.IMAGE_MAX_WIDTH || 1200)
const webpQuality = Number(process.env.IMAGE_WEBP_QUALITY || 76)
const keepOriginalImages = process.env.IMAGE_KEEP_ORIGINALS === "true"

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = await Promise.all(
    entries.map(async (entry) => {
      const fullPath = path.join(dir, entry.name)

      if (entry.isDirectory()) {
        return walk(fullPath)
      }

      return fullPath
    }),
  )

  return files.flat()
}

function toWebpPath(filePath) {
  const extension = path.extname(filePath)
  return filePath.slice(0, -extension.length) + ".webp"
}

function toPosixPath(filePath) {
  return filePath.split(path.sep).join("/")
}

async function optimizeImage(filePath) {
  const outputPath = toWebpPath(filePath)
  const original = await stat(filePath)
  const image = sharp(filePath, { animated: true })
  const metadata = await image.metadata()

  await image
    .rotate()
    .resize({
      width:
        metadata.width && metadata.width > maxImageWidth
          ? maxImageWidth
          : undefined,
      withoutEnlargement: true,
    })
    .webp({
      quality: webpQuality,
      effort: 5,
      smartSubsample: true,
    })
    .toFile(outputPath)

  const optimized = await stat(outputPath)

  return {
    originalPath: filePath,
    outputPath,
    originalBytes: original.size,
    optimizedBytes: optimized.size,
  }
}

async function rewriteReferences(files, conversions) {
  const replacements = new Map(
    conversions.map(({ originalPath, outputPath }) => {
      const originalRelative = toPosixPath(path.relative(distDir, originalPath))
      const outputRelative = toPosixPath(path.relative(distDir, outputPath))

      return [originalRelative, outputRelative]
    }),
  )

  await Promise.all(
    files
      .filter((filePath) => textExtensions.has(path.extname(filePath)))
      .map(async (filePath) => {
        const originalContent = await readFile(filePath, "utf8")
        let content = originalContent

        for (const [from, to] of replacements) {
          content = content.replaceAll(from, to)
          content = content.replaceAll(`/${from}`, `/${to}`)
        }

        if (content !== originalContent) {
          await writeFile(filePath, content)
        }
      }),
  )
}

async function main() {
  const files = await walk(distDir)
  const images = files.filter((filePath) =>
    imageExtensions.has(path.extname(filePath).toLowerCase()),
  )

  if (images.length === 0) {
    console.log("No PNG/JPEG images found in dist.")
    return
  }

  const conversions = await Promise.all(images.map(optimizeImage))
  await rewriteReferences(files, conversions)

  if (!keepOriginalImages) {
    await Promise.all(
      conversions.map(({ originalPath }) => unlink(originalPath)),
    )
  }

  const totalOriginal = conversions.reduce(
    (sum, item) => sum + item.originalBytes,
    0,
  )
  const totalOptimized = conversions.reduce(
    (sum, item) => sum + item.optimizedBytes,
    0,
  )
  const savedPercent =
    totalOriginal > 0
      ? Math.round((1 - totalOptimized / totalOriginal) * 100)
      : 0

  console.log(
    `Optimized ${conversions.length} images to WebP: ${formatBytes(
      totalOriginal,
    )} -> ${formatBytes(totalOptimized)} (${savedPercent}% smaller).${
      keepOriginalImages ? " Original images were kept." : ""
    }`,
  )
}

function formatBytes(bytes) {
  if (bytes < 1024) {
    return `${bytes} B`
  }

  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KiB`
  }

  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})

const optimizableLocalImagePattern =
  /^(\/game-icons\/.+)\.(png|jpe?g)([?#].*)?$/i

export function toOptimizedImageSrc(src?: string) {
  if (!src || !import.meta.env.PROD) return src

  return src.replace(
    optimizableLocalImagePattern,
    (_match, pathname: string, _extension: string, suffix = "") =>
      `${pathname}.webp${suffix}`,
  )
}

export function imageSrcCandidates(src?: string) {
  if (!src) return []

  const optimizedSrc = toOptimizedImageSrc(src)
  const candidates = uniqueImageSrcs([optimizedSrc, src])
  if (
    !import.meta.env.PROD ||
    !src.startsWith("/game-icons/") ||
    src.startsWith("data:") ||
    src.startsWith("blob:")
  ) {
    return candidates
  }

  const retryToken = Date.now().toString(36)
  return uniqueImageSrcs([
    ...candidates,
    ...candidates.map((candidate) =>
      toCacheBustedImageSrc(candidate, retryToken),
    ),
  ])
}

function uniqueImageSrcs(srcs: Array<string | undefined>) {
  return srcs.filter((src, index): src is string =>
    Boolean(src && srcs.indexOf(src) === index),
  )
}

function toCacheBustedImageSrc(src: string, retryToken: string) {
  if (src.startsWith("data:") || src.startsWith("blob:")) return src

  const hashIndex = src.indexOf("#")
  const pathAndQuery = hashIndex >= 0 ? src.slice(0, hashIndex) : src
  const hash = hashIndex >= 0 ? src.slice(hashIndex) : ""
  const separator = pathAndQuery.includes("?") ? "&" : "?"

  return `${pathAndQuery}${separator}cf_retry=${retryToken}${hash}`
}

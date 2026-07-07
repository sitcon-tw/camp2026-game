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

import type { Video } from './api'
// Literal terms, in source order. No frequency, performance score or inferred topics.
const stop = new Set(
  `the a an and or but to of in on at for from with without is are was were be been being this that these those it its my your our their his her as by not no yes how what when where why who you i we they he she me us them have has had do does did can could will would should video videos watch subscribe like share follow channel link links http https www com youtube shorts el la los las un una unos unas de del al en y o pero con sin por para es son fue ser se que como cómo este esta esto ese esa mi mis tu tus su sus nuestro nuestra yo tú nos ellos ellas hay más muy ya si sí no sobre entre desde hasta gracias suscríbete canal ver aquí aqui todos todo cada también también`.split(
    /\s+/,
  ),
)
function unique(values: string[], limit: number) {
  const seen = new Set<string>()
  return values
    .filter((v) => {
      const key = v.toLocaleLowerCase()
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
    .slice(0, limit)
}
export function words(text: string, limit = 24) {
  const clean = text
    .replace(/https?:\/\/\S+|www\.\S+/giu, '')
    .replace(/[#@][\p{L}\p{M}\p{N}_]+/gu, '')
  return unique(
    (clean.match(/[\p{L}][\p{L}\p{M}\p{N}_'-]*/gu) || []).filter(
      (w) => [...w].length > 2 && !stop.has(w.toLocaleLowerCase()),
    ),
    limit,
  )
}
export function sourceTerms(video: Video) {
  return {
    tags: unique(video.tags || [], 50),
    hashtags: unique(
      (video.title + '\n' + video.description).match(/#[\p{L}\p{M}\p{N}_]+/gu) || [],
      30,
    ),
    title: words(video.title),
    description: words(video.description),
  }
}
export function previewTerms(video: Video) {
  const t = sourceTerms(video)
  return unique([...t.tags, ...t.hashtags, ...t.title], 8)
}

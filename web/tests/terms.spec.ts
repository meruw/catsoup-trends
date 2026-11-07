import { test, expect } from '@playwright/test'
import { sourceTerms, words, previewTerms } from '../src/terms'
import type { Video } from '../src/api'
test('literal ES/EN terms preserve source order and exclude links', () => {
  expect(
    words('The shadow and la sombra https://example.com/private #Historia @creator SHADOW'),
  ).toEqual(['shadow', 'sombra'])
  const video = {
    title: 'Una puerta #Animación',
    description: 'Otra historia #Animación #世界 https://example.com',
    tags: ['block story', 'Block Story', 'Minecraft'],
  } as Video
  expect(sourceTerms(video).tags).toEqual(['block story', 'Minecraft'])
  expect(sourceTerms(video).hashtags).toEqual(['#Animación', '#世界'])
  expect(sourceTerms(video).title).toEqual(['puerta'])
  expect(previewTerms(video)).toContain('block story')
  expect(words(Array.from({ length: 100 }, (_, i) => 'term' + i).join(' '))).toHaveLength(24)
})

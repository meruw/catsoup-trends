<script setup lang="ts">
import { computed } from 'vue'
import type { Video } from './api'
import { sourceTerms } from './terms'
const props = defineProps<{ video: Video }>()
const emit = defineEmits<{ search: [term: string] }>()
const groups = computed(() => {
  const t = sourceTerms(props.video)
  return [
    { name: 'Video tags', values: t.tags },
    { name: 'Hashtags', values: t.hashtags },
    { name: 'Title words', values: t.title },
    { name: 'Description words', values: t.description },
  ]
})
</script>
<template>
  <section class="video-terms" aria-label="Source terms">
    <h3>Source terms</h3>
    <p class="meta">From this video's metadata. Select a term to search.</p>
    <div v-for="group in groups" :key="group.name" class="term-group">
      <h4>{{ group.name }}</h4>
      <div v-if="group.values.length" class="term-list">
        <button
          v-for="term in group.values"
          :key="term"
          :title="'Search YouTube for ' + term"
          @click="emit('search', term)"
        >
          {{ term }}
        </button>
      </div>
      <p v-else class="meta">None available</p>
    </div>
    <p class="meta">
      Literal words, not a transcript or a popularity score. Common ES/EN words and links are
      omitted.
    </p>
  </section>
</template>

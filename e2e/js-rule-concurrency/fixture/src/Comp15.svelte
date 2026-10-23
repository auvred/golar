<script lang="ts">
  import { make15, Store15, use15 } from './file15.ts'
  import Comp0 from './Comp0.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store15<'x' | 'y', number>()
  const labels = $derived(use15(make15(count)))
  const doubled = $derived(make15(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp0 start={count} />
{/if}

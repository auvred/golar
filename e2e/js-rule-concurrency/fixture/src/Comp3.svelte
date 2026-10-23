<script lang="ts">
  import { make3, Store3, use3 } from './file3.ts'
  import Comp4 from './Comp4.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store3<'x' | 'y', number>()
  const labels = $derived(use3(make3(count)))
  const doubled = $derived(make3(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp4 start={count} />
{/if}

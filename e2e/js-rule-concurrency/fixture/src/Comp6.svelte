<script lang="ts">
  import { make6, Store6, use6 } from './file6.ts'
  import Comp7 from './Comp7.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store6<'x' | 'y', number>()
  const labels = $derived(use6(make6(count)))
  const doubled = $derived(make6(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp7 start={count} />
{/if}

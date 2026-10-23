<script lang="ts">
  import { make14, Store14, use14 } from './file14.ts'
  import Comp15 from './Comp15.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store14<'x' | 'y', number>()
  const labels = $derived(use14(make14(count)))
  const doubled = $derived(make14(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp15 start={count} />
{/if}

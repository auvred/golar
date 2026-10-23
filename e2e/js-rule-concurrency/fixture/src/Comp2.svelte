<script lang="ts">
  import { make2, Store2, use2 } from './file2.ts'
  import Comp3 from './Comp3.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store2<'x' | 'y', number>()
  const labels = $derived(use2(make2(count)))
  const doubled = $derived(make2(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp3 start={count} />
{/if}

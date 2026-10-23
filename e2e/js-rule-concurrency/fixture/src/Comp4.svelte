<script lang="ts">
  import { make4, Store4, use4 } from './file4.ts'
  import Comp5 from './Comp5.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store4<'x' | 'y', number>()
  const labels = $derived(use4(make4(count)))
  const doubled = $derived(make4(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp5 start={count} />
{/if}

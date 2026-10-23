<script lang="ts">
  import { make7, Store7, use7 } from './file7.ts'
  import Comp8 from './Comp8.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store7<'x' | 'y', number>()
  const labels = $derived(use7(make7(count)))
  const doubled = $derived(make7(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp8 start={count} />
{/if}

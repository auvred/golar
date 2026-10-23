<script lang="ts">
  import { make10, Store10, use10 } from './file10.ts'
  import Comp11 from './Comp11.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store10<'x' | 'y', number>()
  const labels = $derived(use10(make10(count)))
  const doubled = $derived(make10(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp11 start={count} />
{/if}

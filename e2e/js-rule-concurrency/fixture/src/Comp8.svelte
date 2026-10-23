<script lang="ts">
  import { make8, Store8, use8 } from './file8.ts'
  import Comp9 from './Comp9.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store8<'x' | 'y', number>()
  const labels = $derived(use8(make8(count)))
  const doubled = $derived(make8(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp9 start={count} />
{/if}

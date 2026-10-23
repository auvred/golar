<script lang="ts">
  import { make13, Store13, use13 } from './file13.ts'
  import Comp14 from './Comp14.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store13<'x' | 'y', number>()
  const labels = $derived(use13(make13(count)))
  const doubled = $derived(make13(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp14 start={count} />
{/if}

<script lang="ts">
  import { make1, Store1, use1 } from './file1.ts'
  import Comp2 from './Comp2.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store1<'x' | 'y', number>()
  const labels = $derived(use1(make1(count)))
  const doubled = $derived(make1(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp2 start={count} />
{/if}

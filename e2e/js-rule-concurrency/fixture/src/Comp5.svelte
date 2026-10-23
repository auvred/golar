<script lang="ts">
  import { make5, Store5, use5 } from './file5.ts'
  import Comp6 from './Comp6.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store5<'x' | 'y', number>()
  const labels = $derived(use5(make5(count)))
  const doubled = $derived(make5(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp6 start={count} />
{/if}

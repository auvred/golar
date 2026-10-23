<script lang="ts">
  import { make9, Store9, use9 } from './file9.ts'
  import Comp10 from './Comp10.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store9<'x' | 'y', number>()
  const labels = $derived(use9(make9(count)))
  const doubled = $derived(make9(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp10 start={count} />
{/if}

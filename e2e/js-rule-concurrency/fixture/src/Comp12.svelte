<script lang="ts">
  import { make12, Store12, use12 } from './file12.ts'
  import Comp13 from './Comp13.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store12<'x' | 'y', number>()
  const labels = $derived(use12(make12(count)))
  const doubled = $derived(make12(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp13 start={count} />
{/if}

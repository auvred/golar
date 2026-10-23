<script lang="ts">
  import { make0, Store0, use0 } from './file0.ts'
  import Comp1 from './Comp1.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store0<'x' | 'y', number>()
  const labels = $derived(use0(make0(count)))
  const doubled = $derived(make0(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp1 start={count} />
{/if}

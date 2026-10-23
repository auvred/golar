<script lang="ts">
  import { make11, Store11, use11 } from './file11.ts'
  import Comp12 from './Comp12.svelte'

  let { start = 1 }: { start?: number } = $props()
  let count = $state(start)
  const store = new Store11<'x' | 'y', number>()
  const labels = $derived(use11(make11(count)))
  const doubled = $derived(make11(count).map((n) => n * 2).map((n) => `${n}`))
</script>

<button onclick={() => store.set('x', (count += 1))}>{doubled.value}</button>
{#each labels as label (label)}
  <span title={label}>{store.get('x') ?? label.length}</span>
{/each}
{#if count < 0}
  <Comp12 start={count} />
{/if}

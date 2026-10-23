<script lang="ts">
  type Field = { kind: 'draft'; value: string } | { kind: 'saved'; readonly value: string }

  let { field }: { field: Field } = $props()

  function fade(
    node: Element,
    params: { delay: number },
    options: { direction: 'in' | 'out' | 'both' },
  ) {
    return { delay: params.delay, duration: options.direction === 'in' ? 100 : 200 }
  }

  const reported: string = 1
</script>

<input bind:value={field.value} />
<p transition:fade|global={{ delay: 1 }}>x</p>

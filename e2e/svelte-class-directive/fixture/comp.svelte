<script lang="ts">
  let { open }: { open: boolean } = $props()
</script>

<div class="menu" class:is-open={open}>…</div>

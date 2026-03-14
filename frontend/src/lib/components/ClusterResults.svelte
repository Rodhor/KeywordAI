<script lang="ts">
  import { ClipboardIcon, ClipboardCheckIcon } from "@lucide/svelte";

  type ClusterData = {
    header: string[];
    content: string[][];
  };

  let {
    data,
    onReset,
  }: {
    data: ClusterData;
    onReset: () => void;
  } = $props();

  let copied = $state(false);

  function copyAll() {
    const tsv = [data.header, ...data.content]
      .map((row) => row.join("\t"))
      .join("\n");
    navigator.clipboard.writeText(tsv);
    copied = true;
    setTimeout(() => (copied = false), 1001);
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h2 class="h3">Clustering Complete</h2>
    <button type="button" class="btn btn-sm preset-outlined" onclick={copyAll}>
      {#if copied}
        <ClipboardCheckIcon class="size-4" />
        Copied!
      {:else}
        <ClipboardIcon class="size-4" />
        Copy table
      {/if}
    </button>
  </div>

  <div class="table-wrap">
    <table class="table caption-bottom">
      <thead>
        <tr>
          {#each data.header as header}
            <th>{header}</th>
          {/each}
        </tr>
      </thead>
      <tbody class="[&>tr]:hover:preset-tonal-primary">
        {#each data.content as row}
          <tr>
            {#each row as cell}
              <td>{cell}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <button type="button" class="btn preset-outlined" onclick={onReset}>
    Start Over
  </button>
</div>

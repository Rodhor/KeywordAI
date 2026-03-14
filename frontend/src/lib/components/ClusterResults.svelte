<script lang="ts">
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
</script>

<div class="space-y-4">
  <h2 class="h3">Clustering Complete</h2>

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

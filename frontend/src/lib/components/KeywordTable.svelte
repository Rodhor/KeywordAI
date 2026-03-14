<script lang="ts">
  type EvaluationData = {
    headers: string[];
    rows: string[][];
    originalRows: string[][];
  };

  let {
    data,
    onCluster,
    onReset,
  }: {
    data: EvaluationData;
    onCluster: (selectedIndices: number[]) => void;
    onReset: () => void;
  } = $props();

  // Pre-check rows where the Evaluation column equals "Good"
  const evalColIndex = data.headers.findIndex((h) => h === 'Evaluation');

  let checked = $state<boolean[]>(
    data.rows.map((row) =>
      evalColIndex >= 0 ? row[evalColIndex]?.trim() === 'Good' : true
    )
  );

  function toggleAll(value: boolean) {
    checked = checked.map(() => value);
  }

  function handleCluster() {
    const selected = checked
      .map((c, i) => (c ? i : -1))
      .filter((i) => i >= 0);
    onCluster(selected);
  }

  let selectedCount = $derived(checked.filter(Boolean).length);
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h2 class="h3">Evaluation Results</h2>
    <div class="flex gap-2">
      <button type="button" class="btn btn-sm preset-outlined" onclick={() => toggleAll(true)}>
        Select all
      </button>
      <button type="button" class="btn btn-sm preset-outlined" onclick={() => toggleAll(false)}>
        Deselect all
      </button>
    </div>
  </div>

  <div class="table-wrap">
    <table class="table caption-bottom">
      <thead>
        <tr>
          <th class="w-10"></th>
          {#each data.headers as header}
            <th>{header}</th>
          {/each}
        </tr>
      </thead>
      <tbody class="[&>tr]:hover:preset-tonal-primary">
        {#each data.rows as row, i}
          <tr class={checked[i] ? '' : 'opacity-50'}>
            <td>
              <input
                type="checkbox"
                class="checkbox"
                bind:checked={checked[i]}
              />
            </td>
            {#each row as cell}
              <td>{cell}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <div class="flex items-center justify-between">
    <button type="button" class="btn preset-outlined" onclick={onReset}>
      Start Over
    </button>
    <button
      type="button"
      class="btn preset-filled"
      disabled={selectedCount === 0}
      onclick={handleCluster}
    >
      Proceed to Clustering ({selectedCount})
    </button>
  </div>
</div>

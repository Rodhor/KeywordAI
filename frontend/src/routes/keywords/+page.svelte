<script lang="ts">
  import KeywordForm from '$lib/components/KeywordForm.svelte';
  import KeywordTable from '$lib/components/KeywordTable.svelte';
  import ClusterResults from '$lib/components/ClusterResults.svelte';
  import PollingSpinner from '$lib/components/PollingSpinner.svelte';
  import { StartEvaluation, GetEvaluationStatus, StartClustering, GetClusterStatus } from '$lib/wailsjs/go/keywords/KeywordsCommand';

  type Phase = 'form' | 'evaluating' | 'results' | 'clustering' | 'clustered' | 'error';

  type EvaluationData = {
    headers: string[];
    rows: string[][];
    originalRows: string[][];
  };

  type ClusterData = {
    header: string[];
    content: string[][];
  };

  let phase = $state<Phase>('form');
  let evalResult = $state<EvaluationData | null>(null);
  let clusterResult = $state<ClusterData | null>(null);
  let errorMsg = $state('');
  let evalJobId = $state('');

  async function handleSubmit(params: {
    filename: string;
    campaignDesc: string;
    excludeKeywords: string[];
    customInstructions: string;
  }) {
    phase = 'evaluating';
    errorMsg = '';
    try {
      const { jobId } = await StartEvaluation(params);
      evalJobId = jobId;
      poll(jobId, GetEvaluationStatus, (data) => {
        evalResult = data as EvaluationData;
        phase = 'results';
      });
    } catch (e) {
      errorMsg = String(e);
      phase = 'error';
    }
  }

  async function handleCluster(selectedIndices: number[]) {
    phase = 'clustering';
    errorMsg = '';
    try {
      const { jobId } = await StartClustering({ evalJobId, selectedRows: selectedIndices });
      poll(jobId, GetClusterStatus, (data) => {
        const raw = data as { Header: string[]; Content: string[][] };
        clusterResult = { header: raw.Header, content: raw.Content };
        phase = 'clustered';
      });
    } catch (e) {
      errorMsg = String(e);
      phase = 'error';
    }
  }

  function poll(
    jobId: string,
    statusFn: (id: string) => Promise<{ status: string; error?: string; data?: unknown }>,
    onComplete: (data: unknown) => void
  ) {
    const interval = setInterval(async () => {
      try {
        const s = await statusFn(jobId);
        if (s.status === 'completed') {
          clearInterval(interval);
          onComplete(s.data);
        } else if (s.status === 'failed') {
          clearInterval(interval);
          errorMsg = s.error ?? 'Unknown error';
          phase = 'error';
        }
      } catch (e) {
        clearInterval(interval);
        errorMsg = String(e);
        phase = 'error';
      }
    }, 1000);
  }

  function reset() {
    phase = 'form';
    evalResult = null;
    clusterResult = null;
    errorMsg = '';
    evalJobId = '';
  }
</script>

{#if phase === 'form'}
  <KeywordForm onSubmit={handleSubmit} />

{:else if phase === 'evaluating'}
  <div class="flex justify-center pt-24">
    <PollingSpinner message="Evaluating keywords..." />
  </div>

{:else if phase === 'results' && evalResult}
  <KeywordTable data={evalResult} onCluster={handleCluster} onReset={reset} />

{:else if phase === 'clustering'}
  <div class="flex justify-center pt-24">
    <PollingSpinner message="Clustering keywords..." />
  </div>

{:else if phase === 'clustered' && clusterResult}
  <ClusterResults data={clusterResult} onReset={reset} />

{:else if phase === 'error'}
  <div class="mx-auto max-w-xl space-y-4">
    <div class="card preset-outlined-error-500 p-6 space-y-2">
      <h2 class="h3 text-error-500">Something went wrong</h2>
      <p class="text-sm">{errorMsg}</p>
    </div>
    <button type="button" class="btn preset-outlined" onclick={reset}>
      Start Over
    </button>
  </div>
{/if}

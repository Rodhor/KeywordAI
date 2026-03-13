<script lang="ts">
  import { Progress } from '@skeletonlabs/skeleton-svelte';

  let { message = 'Processing...' }: { message?: string } = $props();
</script>

<div class="flex flex-col items-center gap-4">
  <Progress value={null} class="items-center w-fit">
    <Progress.Circle>
      <Progress.CircleTrack />
      <Progress.CircleRange />
    </Progress.Circle>
  </Progress>
  <p class="text-surface-600-400">{message}</p>
</div>

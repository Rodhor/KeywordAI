<script lang="ts">
  import { TagsInput } from '@skeletonlabs/skeleton-svelte';
  import { PickAndSaveCSV } from '$lib/wailsjs/go/upload/UploadCommand';

  type FormData = {
    filename: string;
    campaignDesc: string;
    excludeKeywords: string[];
    customInstructions: string;
  };

  let { onSubmit }: { onSubmit: (data: FormData) => void } = $props();

  let filename = $state('');
  let uploadError = $state('');
  let campaignDesc = $state('');
  let excludeKeywords = $state<string[]>([]);
  let customInstructions = $state('');

  async function pickFile() {
    const [result, err] = await PickAndSaveCSV().then(
      (r) => [r, null] as const,
      (e) => [null, String(e)] as const
    );

    if (err) {
      uploadError = err;
      filename = '';
    } else if (result) {
      filename = result;
      uploadError = '';
    }
    // result === '' means user cancelled — leave state unchanged
  }

  function handleSubmit() {
    onSubmit({ filename, campaignDesc, excludeKeywords, customInstructions });
  }
</script>

<div class="mx-auto max-w-xl space-y-6">
  <h1 class="h2">New Campaign</h1>

  <!-- File picker -->
  <div class="space-y-2">
    <span class="label-text">Keyword CSV</span>
    <div class="flex items-center gap-3">
      <button type="button" class="btn preset-outlined" onclick={pickFile}>
        {filename ? filename : 'Choose CSV file'}
      </button>
      {#if filename}
        <span class="badge preset-filled-success-500">Ready</span>
      {/if}
    </div>
    {#if uploadError}
      <p class="text-error-500 text-sm">{uploadError}</p>
    {/if}
  </div>

  <!-- Campaign description -->
  <label class="label">
    <span class="label-text">Campaign Description</span>
    <textarea
      class="textarea rounded-container"
      rows="3"
      placeholder="Describe the campaign so the AI can evaluate keywords in context..."
      bind:value={campaignDesc}
    ></textarea>
  </label>

  <!-- Exclude keywords -->
  <div class="space-y-1">
    <TagsInput value={excludeKeywords} onValueChange={(e) => (excludeKeywords = e.value)}>
      <TagsInput.Label>Exclude Keywords</TagsInput.Label>
      <TagsInput.Control>
        <TagsInput.Context>
          {#snippet children(tagsInput)}
            {#each tagsInput().value as value, index (index)}
              <TagsInput.Item {value} {index}>
                <TagsInput.ItemPreview class="preset-filled-secondary-500">
                  <TagsInput.ItemText>{value}</TagsInput.ItemText>
                  <TagsInput.ItemDeleteTrigger />
                </TagsInput.ItemPreview>
                <TagsInput.ItemInput />
              </TagsInput.Item>
            {/each}
          {/snippet}
        </TagsInput.Context>
        <TagsInput.Input placeholder="Type a keyword and press Enter..." />
      </TagsInput.Control>
      <TagsInput.HiddenInput />
    </TagsInput>
  </div>

  <!-- Custom instructions -->
  <label class="label">
    <span class="label-text">Custom Instructions <span class="text-surface-400">(optional)</span></span>
    <textarea
      class="textarea rounded-container"
      rows="2"
      placeholder="Any extra guidance for the AI evaluator..."
      bind:value={customInstructions}
    ></textarea>
  </label>

  <!-- Submit -->
  <button
    type="button"
    class="btn preset-filled w-full"
    disabled={!filename || !campaignDesc}
    onclick={handleSubmit}
  >
    Evaluate Keywords
  </button>
</div>

<script lang="ts">
  import { SearchIcon, BriefcaseIcon } from '@lucide/svelte';
</script>

<div class="max-w-3xl mx-auto space-y-8">
  <div class="space-y-1">
    <h1 class="h1">Tools</h1>
    <p class="text-surface-500">Select a tool to get started.</p>
  </div>

  <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">

    <!-- Keyword Evaluator -->
    <a href="/keywords" class="card preset-outlined-surface-200-800 bg-surface-50-950 p-6 space-y-3 hover:preset-outlined-primary-500 transition-all block">
      <div class="flex items-center justify-between">
        <div class="p-2 rounded-lg bg-primary-500/10">
          <SearchIcon class="size-6 text-primary-500" />
        </div>
        <span class="badge preset-filled-success-500 text-xs">Active</span>
      </div>
      <div class="space-y-1">
        <h2 class="h4">Keyword Evaluator</h2>
        <p class="text-sm text-surface-500">Upload a keyword CSV, evaluate relevance with AI, and cluster results for your campaign.</p>
      </div>
    </a>

    <!-- Campaign Setup — placeholder -->
    <div class="card preset-outlined-surface-200-800 bg-surface-50-950 p-6 space-y-3 opacity-50 cursor-not-allowed">
      <div class="flex items-center justify-between">
        <div class="p-2 rounded-lg bg-surface-500/10">
          <BriefcaseIcon class="size-6 text-surface-400" />
        </div>
        <span class="badge preset-tonal text-xs">Coming Soon</span>
      </div>
      <div class="space-y-1">
        <h2 class="h4">Campaign Setup</h2>
        <p class="text-sm text-surface-500">Create a Google Drive folder, Docs brief, and Sheets export — all from one form.</p>
      </div>
    </div>

  </div>
</div>

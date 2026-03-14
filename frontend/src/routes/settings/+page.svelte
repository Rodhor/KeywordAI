<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { EyeIcon, EyeOffIcon } from "@lucide/svelte";
  import { GetSettings, Save } from "$lib/wailsjs/go/settings/SettingsCommand";

  let geminiKey = $state("");
  let existingGeminiKey = $state("");
  let googleCredentials = $state("");
  let showKey = $state(false);
  let saving = $state(false);
  let errorMsg = $state("");

  onMount(async () => {
    try {
      const s = await GetSettings();
      existingGeminiKey = s.GeminiAPIKey ?? "";
      googleCredentials = s.GoogleAuthAPIKey ?? "";
    } catch (e) {
      errorMsg = String(e);
    }
  });

  async function handleSave() {
    saving = true;
    errorMsg = "";
    try {
      await Save({
        GeminiAPIKey: geminiKey || existingGeminiKey,
        GoogleAuthAPIKey: googleCredentials,
      });
      await goto("/");
    } catch (e) {
      errorMsg = String(e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="mx-auto max-w-xl space-y-6">
  <h1 class="h2">Settings</h1>

  <!-- Gemini API Key -->
  <label class="label">
    <span class="label-text">Gemini API Key</span>
    <div class="relative flex items-center">
      <input
        class="input rounded-container pr-10 w-full"
        type={showKey ? "text" : "password"}
        placeholder={existingGeminiKey
          ? "API key saved — enter a new key to replace it"
          : "Enter your Gemini API key"}
        bind:value={geminiKey}
        required
      />
      <button
        type="button"
        class="btn-icon absolute right-2 hover:preset-tonal"
        aria-label={showKey ? "Hide key" : "Show key"}
        onclick={() => (showKey = !showKey)}
      >
        {#if showKey}
          <EyeOffIcon class="size-4" />
        {:else}
          <EyeIcon class="size-4" />
        {/if}
      </button>
    </div>
  </label>

  <!-- Google Credentials JSON (placeholder) -->
  <div class="space-y-1">
    <div class="flex items-center gap-2">
      <span class="label-text">Google Credentials JSON</span>
      <span class="badge preset-outlined-surface-500 text-xs">Coming Soon</span>
    </div>
    <textarea
      class="textarea rounded-container opacity-50 cursor-not-allowed"
      rows="4"
      placeholder="Will be needed for Google Sheets export..."
      disabled
    ></textarea>
  </div>

  <!-- Feedback -->
  {#if errorMsg}
    <p class="text-error-500 text-sm">{errorMsg}</p>
  {/if}

  <!-- Save -->
  <button
    type="button"
    class="btn preset-filled w-full"
    disabled={saving || (!geminiKey && !existingGeminiKey)}
    onclick={handleSave}
  >
    {saving ? "Saving…" : "Save Settings"}
  </button>
</div>

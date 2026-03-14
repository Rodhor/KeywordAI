<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { EyeIcon, EyeOffIcon } from "@lucide/svelte";
  import {
    Combobox,
    Portal,
    useListCollection,
    type ComboboxRootProps,
  } from "@skeletonlabs/skeleton-svelte";
  import { GetSettings, Save } from "$lib/wailsjs/go/settings/SettingsCommand";

  let geminiKey = $state("");
  let existingGeminiKey = $state("");
  let googleCredentials = $state("");
  let showKey = $state(false);
  let saving = $state(false);
  let errorMsg = $state("");

  const allThemes = [
    { value: "catppuccin", label: "Catppuccin" },
    { value: "cerberus", label: "Cerberus" },
    { value: "concord", label: "Concord" },
    { value: "crimson", label: "Crimson" },
    { value: "fennec", label: "Fennec" },
    { value: "hamlindigo", label: "Hamlindigo" },
    { value: "mint", label: "Mint" },
    { value: "modern", label: "Modern" },
    { value: "mona", label: "Mona" },
    { value: "nosh", label: "Nosh" },
    { value: "nouveau", label: "Nouveau" },
    { value: "pine", label: "Pine" },
    { value: "reign", label: "Reign" },
    { value: "rocket", label: "Rocket" },
    { value: "rose", label: "Rose" },
    { value: "sahara", label: "Sahara" },
    { value: "seafoam", label: "Seafoam" },
    { value: "terminus", label: "Terminus" },
    { value: "vintage", label: "Vintage" },
    { value: "vox", label: "Vox" },
    { value: "wintry", label: "Wintry" },
  ];

  let themeItems = $state(allThemes);

  const themeCollection = $derived(
    useListCollection({
      items: themeItems,
      itemToString: (item) => item.label,
      itemToValue: (item) => item.value,
    }),
  );

  let selectedTheme = $state(localStorage.getItem("theme") ?? "catppuccin");

  function applyTheme(value: string) {
    selectedTheme = value;
    document.documentElement.setAttribute("data-theme", value);
    localStorage.setItem("theme", value);
  }

  const onThemeOpenChange = () => {
    themeItems = allThemes;
  };

  const onThemeInputValueChange: ComboboxRootProps["onInputValueChange"] = (
    event,
  ) => {
    const filtered = allThemes.filter((item) =>
      item.label.toLowerCase().includes(event.inputValue.toLowerCase()),
    );
    themeItems = filtered.length > 0 ? filtered : allThemes;
  };

  const onThemeValueChange: ComboboxRootProps["onValueChange"] = (event) => {
    if (event.value[0]) applyTheme(event.value[0]);
  };

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

  <!-- Appearance -->
  <div
    class="card preset-outlined-surface-200-800 bg-surface-50-950 p-6 space-y-4"
  >
    <h2 class="h4">Appearance</h2>

    <Combobox
      value={[selectedTheme]}
      collection={themeCollection}
      onOpenChange={onThemeOpenChange}
      onInputValueChange={onThemeInputValueChange}
      onValueChange={onThemeValueChange}
      placeholder="Search themes..."
    >
      <Combobox.Label class="label-text">Theme</Combobox.Label>
      <Combobox.Control class="input-group grid-cols-[1fr_auto]">
        <Combobox.Input class="ig-input" />
        <Combobox.Trigger class="ig-btn preset-tonal" />
      </Combobox.Control>
      <Portal>
        <Combobox.Positioner>
          <Combobox.Content
            class="card bg-surface-100-900 border border-surface-200-800 p-1 space-y-0.5 overflow-y-auto shadow-xl"
          >
            {#each themeItems as item (item.value)}
              <Combobox.Item
                {item}
                class="px-3 py-1.5 rounded cursor-pointer hover:preset-tonal data-highlighted:preset-tonal"
              >
                <Combobox.ItemText>{item.label}</Combobox.ItemText>
                <Combobox.ItemIndicator>✓</Combobox.ItemIndicator>
              </Combobox.Item>
            {/each}
          </Combobox.Content>
        </Combobox.Positioner>
      </Portal>
    </Combobox>
  </div>

  <!-- API Keys -->
  <div
    class="card preset-outlined-surface-200-800 bg-surface-50-950 p-6 space-y-4"
  >
    <h2 class="h4">API Keys</h2>

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
  </div>

  <!-- Data Sources -->
  <div
    class="card preset-outlined-surface-200-800 bg-surface-50-950 p-6 space-y-4"
  >
    <div class="flex items-center gap-3">
      <h2 class="h4">Data Sources</h2>
      <span class="badge preset-outlined-surface-500 text-xs">Coming Soon</span>
    </div>

    <div class="space-y-1">
      <span class="label-text">Google Credentials JSON</span>
      <textarea
        class="textarea rounded-container opacity-50 cursor-not-allowed"
        rows="4"
        placeholder="Will be needed for Google Sheets export..."
        disabled
      ></textarea>
    </div>
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

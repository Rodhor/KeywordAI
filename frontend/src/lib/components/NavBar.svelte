<script lang="ts">
  import { AppBar } from "@skeletonlabs/skeleton-svelte";
  import { ArrowLeftIcon } from "@lucide/svelte";
  import { page } from "$app/state";
  import LightSwitch from "./LightSwitch.svelte";
  let isHome = $derived(page.url.pathname === "/");
</script>

<AppBar>
  <AppBar.Toolbar class="grid-cols-[auto_1fr]">
    <AppBar.Lead>
      {#if !isHome}
        <a href="/" class="btn-icon hover:preset-tonal" aria-label="Home">
          <ArrowLeftIcon class="size-5" />
        </a>
      {/if}
    </AppBar.Lead>
    <AppBar.Headline>
      <a
        href="/"
        class="text-xl font-semibold hover:opacity-80 transition-opacity"
        >KeywordAI</a
      >
    </AppBar.Headline>
    <AppBar.Trail>
      <LightSwitch />
    </AppBar.Trail>
  </AppBar.Toolbar>
</AppBar>

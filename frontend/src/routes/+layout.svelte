<script lang="ts">
  import "./layout.css";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import NavBar from "$lib/components/NavBar.svelte";
  import { HasRequiredConfig } from "$lib/wailsjs/go/settings/SettingsCommand";

  let { children } = $props();

  $effect(() => {
    HasRequiredConfig()
      .then((ok) => {
        if (!ok && (page.url.pathname as string) !== '/settings') {
          goto('/settings');
        }
      })
      .catch(() => {
        if ((page.url.pathname as string) !== '/settings') {
          goto('/settings');
        }
      });
  });
</script>

<div class="h-screen flex flex-col">
  <NavBar />
  <main class="flex-1 overflow-auto p-8">
    {@render children()}
  </main>
</div>

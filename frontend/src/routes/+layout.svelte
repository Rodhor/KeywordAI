<script lang="ts">
  import "./layout.css";
  import NavBar from "$lib/components/NavBar.svelte";

  let { children } = $props();
</script>

<div class="h-screen flex flex-col">
  <NavBar />
  <main class="flex-1 overflow-auto p-8">
    {@render children()}
  </main>
</div>

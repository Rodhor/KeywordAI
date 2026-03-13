<script lang="ts">
  import { SunIcon, MoonIcon } from '@lucide/svelte';

  // Read from the DOM attribute already set by the inline script below —
  // avoids the $effect mismatch that caused the "switch twice" bug.
  let isDark = $state(
    document.documentElement.getAttribute('data-mode') !== 'light'
  );

  function toggle() {
    isDark = !isDark;
    const mode = isDark ? 'dark' : 'light';
    document.documentElement.setAttribute('data-mode', mode);
    localStorage.setItem('mode', mode);
  }
</script>

<!-- Sets data-mode before the component renders, preventing a theme flash -->
<svelte:head>
  <script>
    document.documentElement.setAttribute(
      'data-mode',
      localStorage.getItem('mode') || 'dark'
    );
  </script>
</svelte:head>

<button type="button" class="btn-icon hover:preset-tonal" onclick={toggle} aria-label="Toggle theme">
  {#if isDark}
    <SunIcon class="size-5" />
  {:else}
    <MoonIcon class="size-5" />
  {/if}
</button>

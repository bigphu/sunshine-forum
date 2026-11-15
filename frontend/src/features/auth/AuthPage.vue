<script setup lang="ts">
// Trang tài khoản không có khung app: logo, một thẻ giữa trang nền nhạt (prototype auP)
defineProps<{ title: string }>()
</script>

<template>
  <main id="mc" class="au" tabindex="-1">
    <div class="auw">
      <RouterLink class="logo" to="/" aria-label="Sunshine Apartment Forum, trang chủ">
        <img src="/brand/sunshine-lockup-horizontal-on-light.svg" alt="" />
      </RouterLink>
      <section class="card auc" aria-labelledby="focus-t">
        <h1 id="focus-t" tabindex="-1">{{ title }}</h1>
        <slot />
      </section>
      <p class="au-f">Diễn đàn cư dân Sunshine Apartment</p>
    </div>
  </main>
</template>

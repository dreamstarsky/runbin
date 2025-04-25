<template>
  <div class="flex flex-col h-screen">
    <NavBar />
    <router-view :key="$route.fullPath" />
  </div>
</template>
<script setup>
import NavBar from './components/NavBar.vue';
</script>
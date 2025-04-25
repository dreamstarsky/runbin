<template>
    <div class="bg-black text-lg flex flex-row **:items-center p-4">
        <div class="w-10 h-10 rounded-full overflow-hidden">
            <img :src="MeowLogo" alt="Logo">
        </div>
        <div class="text-white ml-4 font-bold text-xl leading-10">
            Meow Paste
        </div>
    </div>
</template>
<script setup>
import MeowLogo from '../assets/meow.png'
</script>
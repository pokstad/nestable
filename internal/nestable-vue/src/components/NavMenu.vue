<script setup>
defineProps({
  msg: {
    type: String,
    required: false
  }
})
</script>

<template>
    <nav>
        <ul>
            <li><strong>Nestable</strong></li>
        </ul>
        <ul>
            <li><a href="#TODO">Notes</a></li>
            <li><a href="#TODO">Word Cloud</a></li>
        </ul>
        
    </nav>
</template>

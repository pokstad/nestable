<script setup>
defineProps({
  notes: {}
})
</script>

<template>
    <div class="container-fluid">
        <div v-for="note in notes" :key="note.ID">
            <hr/>
            <small align="right"><p>
                <b>[{{ note.ID }}]</b> - 
                <em>{{ note.Timestamp }}</em>
            </p></small>
            <p align="left">{{ note.Header }}</p>
        </div>
    </div>
</template>

<script setup>
import NavMenu from './components/NavMenu.vue'
import NotesList from './components/NotesList.vue'
</script>

<script>
export default {
  data: () => ({
    notes: null
  }),

  created() {
    // fetch on init
    this.fetchData()
  },

  mounted() {
    this.focusInput();
  },
  
  methods: {
    focusInput() {
      this.$refs.search.focus();
    },
    async fetchData() {
      const url = `/notes`
      this.notes = await (await fetch(url)).json()
    }
  }
}
</script>

<template>
    <main class="container-fluid">
        <NavMenu/>
        <input type="search" ref="search" placeholder="Fuzzy search note headers" />
        <NotesList :notes=notes />
    </main>
</template>

<style scoped>
.content {
  padding: 25px;
}
</style>

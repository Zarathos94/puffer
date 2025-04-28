<script setup>
import RateChart from './components/RateChart.vue'
</script>

<template>
  <div class="center-viewport">
    <div class="center-inner">
      <RateChart />
    </div>
  </div>
</template>

<style scoped>
.center-viewport {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
}
.center-inner {
  max-width: 800px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

header {
  line-height: 1.5;
}

.logo {
  display: block;
  margin: 0 auto 2rem;
}

@media (min-width: 1024px) {
  header {
    display: flex;
    place-items: center;
    padding-right: calc(var(--section-gap) / 2);
  }

  .logo {
    margin: 0 2rem 0 0;
  }

  header .wrapper {
    display: flex;
    place-items: flex-start;
    flex-wrap: wrap;
  }
}
</style>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="status-bar" aria-hidden="true"></div>

      <header class="card-head">
        <div class="tunnel-mark" aria-hidden="true">
          <span class="node"></span>
          <span class="link"></span>
          <span class="node"></span>
        </div>
        <h1 class="brand">G-Tunnel</h1>
        <p class="tagline">WS / HTTP / TCP Relay</p>
      </header>

      <div class="form-head">
        <h2>Sign in</h2>
      </div>

      <LoginForm />
    </div>
  </div>
</template>

<script setup lang="ts" name="login">
import LoginForm from "./components/LoginForm.vue";
</script>

<style scoped lang="scss">
@import "./index.scss";
</style>

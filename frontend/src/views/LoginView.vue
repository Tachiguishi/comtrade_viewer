<template>
  <div class="login-page">
    <n-card title="登录" :bordered="false" class="login-card">
      <n-alert v-if="error" type="error" class="login-alert">{{ error }}</n-alert>
      <n-form @submit.prevent="handleLogin">
        <n-form-item label="用户名">
          <n-input v-model:value="username" placeholder="请输入用户名" autocomplete="username" />
        </n-form-item>
        <n-form-item label="密码">
          <n-input
            v-model:value="password"
            type="password"
            show-password-on="click"
            placeholder="请输入密码"
            autocomplete="current-password"
          />
        </n-form-item>
        <n-button block type="primary" :loading="loading" @click="handleLogin">登录</n-button>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { extractApiError } from '@/api'

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

const message = useMessage()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const handleLogin = async () => {
  loading.value = true
  error.value = ''
  try {
    await authStore.login(username.value, password.value)
    message.success('登录成功')
    const redirect = (route.query.redirect as string) || '/upload'
    router.replace(redirect)
  } catch (err: unknown) {
    const { message: msg } = extractApiError(err)
    error.value = msg
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 20% 20%, rgba(82, 129, 236, 0.2), transparent 25%),
    radial-gradient(circle at 80% 0%, rgba(255, 255, 255, 0.15), transparent 30%),
    linear-gradient(135deg, #0f172a, #1e293b);
  padding: 24px;
}

.login-card {
  width: 360px;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.18);
  border-radius: 16px;
}

.login-alert {
  margin-bottom: 12px;
}
</style>

<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'

definePageMeta({ layout: false })

const { login, getMenu } = useAuth()

const form = ref({
  username: '',
  password: '',
  tenant_id: 'default',
})
const loading = ref(false)

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    message.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const ok = await login(form.value)
    if (ok) {
      await getMenu()
      message.success('登录成功')
      navigateTo('/dashboard')
    } else {
      message.error('登录失败，请检查凭证')
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div style="display: flex; justify-content: center; align-items: center; min-height: 100vh; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
    <a-card style="width: 420px; border-radius: 12px; box-shadow: 0 8px 32px rgba(0,0,0,0.15);">
      <div style="text-align: center; margin-bottom: 24px;">
        <h2 style="margin: 0; font-size: 24px; color: #1a1a2e;">OA智审</h2>
        <p style="color: #888; margin-top: 4px;">流程智能审核平台</p>
      </div>
      <a-form layout="vertical" @finish="handleLogin">
        <a-form-item label="用户名">
          <a-input v-model:value="form.username" placeholder="请输入用户名" size="large" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="form.password" placeholder="请输入密码" size="large" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" block size="large" :loading="loading">
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons-vue'

definePageMeta({ layout: false })

const { login, getMenu } = useAuth()

const form = ref({
  username: '',
  password: '',
  tenant_id: 'default',
})
const loading = ref(false)
const rememberMe = ref(false)

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
      message.success('登录成功，正在跳转...')
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
  <div class="login-page">
    <!-- Animated background -->
    <div class="login-bg">
      <div class="login-bg-shape login-bg-shape--1" />
      <div class="login-bg-shape login-bg-shape--2" />
      <div class="login-bg-shape login-bg-shape--3" />
    </div>

    <div class="login-container">
      <!-- Left: Branding -->
      <div class="login-branding">
        <div class="login-branding-content">
          <div class="login-logo">
            <SafetyCertificateOutlined class="login-logo-icon" />
          </div>
          <h1 class="login-brand-title">OA智审</h1>
          <p class="login-brand-subtitle">流程智能审核平台</p>
          <div class="login-features">
            <div class="login-feature-item">
              <span class="login-feature-dot" />
              <span>AI 驱动的智能审批辅助</span>
            </div>
            <div class="login-feature-item">
              <span class="login-feature-dot" />
              <span>多维度规则自动校验</span>
            </div>
            <div class="login-feature-item">
              <span class="login-feature-dot" />
              <span>全流程可追溯审计</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: Login Form -->
      <div class="login-form-wrapper">
        <div class="login-form-inner">
          <div class="login-form-header">
            <h2>欢迎回来</h2>
            <p>请登录您的账户以继续</p>
          </div>

          <a-form layout="vertical" @finish="handleLogin" class="login-form">
            <a-form-item>
              <a-input
                v-model:value="form.username"
                placeholder="请输入用户名"
                size="large"
                class="login-input"
              >
                <template #prefix>
                  <UserOutlined class="login-input-icon" />
                </template>
              </a-input>
            </a-form-item>

            <a-form-item>
              <a-input-password
                v-model:value="form.password"
                placeholder="请输入密码"
                size="large"
                class="login-input"
              >
                <template #prefix>
                  <LockOutlined class="login-input-icon" />
                </template>
              </a-input-password>
            </a-form-item>

            <div class="login-options">
              <a-checkbox v-model:checked="rememberMe">记住登录</a-checkbox>
            </div>

            <a-form-item>
              <a-button
                type="primary"
                html-type="submit"
                block
                size="large"
                :loading="loading"
                class="login-btn"
              >
                {{ loading ? '登录中...' : '登 录' }}
              </a-button>
            </a-form-item>
          </a-form>

          <div class="login-footer">
            <span>OA智审 © 2025</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Mobile branding (shown only on small screens) -->
    <div class="login-mobile-brand">
      <SafetyCertificateOutlined class="login-mobile-logo" />
      <span>OA智审</span>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: #0f172a;
}

/* Animated background */
.login-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.login-bg-shape {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.5;
  animation: float 20s ease-in-out infinite;
}

.login-bg-shape--1 {
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  top: -200px;
  left: -100px;
  animation-delay: 0s;
}

.login-bg-shape--2 {
  width: 500px;
  height: 500px;
  background: linear-gradient(135deg, #06b6d4, #3b82f6);
  bottom: -150px;
  right: -100px;
  animation-delay: -7s;
}

.login-bg-shape--3 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, #8b5cf6, #ec4899);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  animation-delay: -14s;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  25% { transform: translate(30px, -30px) scale(1.05); }
  50% { transform: translate(-20px, 20px) scale(0.95); }
  75% { transform: translate(20px, 10px) scale(1.02); }
}

/* Container */
.login-container {
  position: relative;
  z-index: 1;
  display: flex;
  width: 900px;
  max-width: calc(100vw - 32px);
  min-height: 540px;
  border-radius: 24px;
  overflow: hidden;
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.4);
}

/* Left branding panel */
.login-branding {
  flex: 1;
  background: linear-gradient(135deg, rgba(79, 70, 229, 0.9), rgba(124, 58, 237, 0.9));
  backdrop-filter: blur(20px);
  padding: 48px 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-branding::before {
  content: '';
  position: absolute;
  inset: 0;
  background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.05'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
}

.login-branding-content {
  position: relative;
  z-index: 1;
}

.login-logo {
  width: 64px;
  height: 64px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.login-logo-icon {
  font-size: 32px;
  color: #fff;
}

.login-brand-title {
  font-size: 32px;
  font-weight: 700;
  color: #fff;
  margin: 0 0 8px;
  letter-spacing: -0.02em;
}

.login-brand-subtitle {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.8);
  margin: 0 0 40px;
}

.login-features {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.login-feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.9);
  font-size: 14px;
}

.login-feature-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22d3ee;
  flex-shrink: 0;
}

/* Right form panel */
.login-form-wrapper {
  flex: 1;
  background: #ffffff;
  padding: 48px 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.login-form-inner {
  max-width: 340px;
  width: 100%;
  margin: 0 auto;
}

.login-form-header {
  margin-bottom: 32px;
}

.login-form-header h2 {
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  margin: 0 0 8px;
}

.login-form-header p {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

.login-form :deep(.ant-form-item) {
  margin-bottom: 20px;
}

.login-input {
  height: 48px !important;
  border-radius: 12px !important;
  border: 1.5px solid #e2e8f0 !important;
  background: #f8fafc !important;
  font-size: 15px !important;
  transition: all 0.2s ease !important;
}

.login-input:hover {
  border-color: #cbd5e1 !important;
}

.login-input:focus,
.login-input-focused {
  border-color: #4f46e5 !important;
  background: #fff !important;
  box-shadow: 0 0 0 4px rgba(79, 70, 229, 0.08) !important;
}

:deep(.ant-input-affix-wrapper:focus),
:deep(.ant-input-affix-wrapper-focused) {
  border-color: #4f46e5 !important;
  background: #fff !important;
  box-shadow: 0 0 0 4px rgba(79, 70, 229, 0.08) !important;
}

.login-input-icon {
  color: #94a3b8;
  font-size: 16px;
}

.login-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.login-btn {
  height: 48px !important;
  border-radius: 12px !important;
  font-size: 16px !important;
  font-weight: 600 !important;
  background: linear-gradient(135deg, #4f46e5, #7c3aed) !important;
  border: none !important;
  box-shadow: 0 4px 16px rgba(79, 70, 229, 0.35) !important;
  transition: all 0.3s ease !important;
}

.login-btn:hover {
  box-shadow: 0 6px 24px rgba(79, 70, 229, 0.45) !important;
  transform: translateY(-1px) !important;
}

.login-btn:active {
  transform: translateY(0) !important;
}

.login-footer {
  text-align: center;
  margin-top: 32px;
  color: #94a3b8;
  font-size: 13px;
}

/* Mobile branding */
.login-mobile-brand {
  display: none;
  position: absolute;
  top: 24px;
  left: 24px;
  z-index: 2;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  align-items: center;
  gap: 8px;
}

.login-mobile-logo {
  font-size: 24px;
}

/* Responsive */
@media (max-width: 768px) {
  .login-branding {
    display: none;
  }

  .login-container {
    min-height: auto;
    border-radius: 20px;
  }

  .login-form-wrapper {
    padding: 40px 24px;
    border-radius: 20px;
  }

  .login-mobile-brand {
    display: flex;
  }

  .login-form-header {
    text-align: center;
    margin-bottom: 28px;
  }
}

@media (max-width: 480px) {
  .login-container {
    max-width: calc(100vw - 24px);
  }

  .login-form-wrapper {
    padding: 32px 20px;
  }
}
</style>

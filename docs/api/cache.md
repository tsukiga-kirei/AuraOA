# 缓存管理接口

> 权限要求：JWT + TenantContext + `system_admin` 角色
>
> 路由前缀：`/api/admin/cache`

## 获取缓存统计

```
GET /api/admin/cache/stats
```

返回 Redis 缓存的命中率、键数量、内存占用等统计信息。

---

## 清除租户缓存

```
DELETE /api/admin/cache/tenant/:tenant_id
```

清除指定租户的所有缓存数据。

---

## 清除模块缓存

```
DELETE /api/admin/cache/module/:module
```

清除指定功能模块的缓存数据。

---

## 切换缓存开关

```
POST /api/admin/cache/toggle
```

全局启用或禁用缓存功能。

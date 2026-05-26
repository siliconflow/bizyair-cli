# BizyAir Web API 文档

> 本文档通过浏览器抓包整理，用于 CLI 工具集成参考。
> 基础域名：`https://bizyair.cn`

---

## 通用说明

### 认证方式

所有需要认证的接口在请求头中携带 Bearer Token：

```
Authorization: Bearer v4.public.{paseto_token}
```

Token 为 PASETO v4 格式，包含 `user_id`、`exp`、`iat`、`iss` ("BizyAir") 等字段。

> 下方 curl 示例中 `$TOKEN` 环境变量需替换为实际的 Bearer Token。

### 通用响应格式

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 状态码，20000=成功 |
| `message` | string | 状态描述 |
| `status` | bool | 是否成功 |
| `data` | object | 业务数据 |

> 金额字段通常提供两种格式：可读格式（如 `"65.3w"`）和精确数值（如 `653633`，后缀 `_amount`）。

### API 路径规则

- `/api/x/v1/` — 通用业务接口
- `/api/y/v1/` — 钱包/交易相关接口

---

## 一、用户模块

### 1.1 获取当前用户元信息

```
GET /api/x/v1/user/metadata
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/user/metadata' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "metadata": {
      "id": "clxfgld8s0005y2roku961vu1",
      "name": "开源爱好者",
      "share_id": "clxfgld8s0005y2roku961vu1",
      "status": "normal",
      "level": 20,
      "last_share_id_update_at": "2024-10-14 16:48:06",
      "avatar": "https://storage.bizyair.cn/web/xxx.webp",
      "introduction": "我是一个客服",
      "auth": 1,
      "auth_type": 1,
      "sub_expire_at": {
        "20": "2026-06-25"
      },
      "third_party_binds": {
        "discord": false,
        "github": false,
        "google": false,
        "h5wechat": false,
        "phone": false,
        "siliconcloud": true,
        "wechat": false
      },
      "online_tester": true,
      "user_level_str": "专业版会员"
    },
    "counter": {
      "api_called_count": 5265,
      "model_used_count": 353377,
      "liked_count": 99,
      "forked_count": 940,
      "workflow_downloaded_count": 4812,
      "follower_count": 36,
      "following_count": 15
    }
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 用户唯一 ID |
| `name` | string | 用户昵称 |
| `level` | int | 会员等级 |
| `user_level_str` | string | 会员等级文字描述 |
| `auth` | int | 实名认证状态（0=未认证, 1=已认证） |
| `auth_type` | int | 认证类型 |
| `sub_expire_at` | object | 各等级到期时间，key 为等级值 |
| `third_party_binds` | object | 第三方账号绑定状态 |
| `counter.api_called_count` | int | API 调用次数 |
| `counter.model_used_count` | int | 模型使用次数 |

---

### 1.2 获取用户空间详情

```
GET /api/x/v1/userspace/{user_id}/detail
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | 用户 ID |

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/userspace/clxfgld8s0005y2roku961vu1/detail' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "id": "clxfgld8s0005y2roku961vu1",
    "name": "开源爱好者",
    "share_id": "clxfgld8s0005y2roku961vu1",
    "status": "normal",
    "level": 20,
    "avatar": "https://storage.bizyair.cn/web/xxx.webp",
    "introduction": "我是一个客服",
    "auth": 1,
    "auth_type": 1,
    "sub_expire_at": { "20": "2026-06-25" },
    "third_party_binds": { ... },
    "online_tester": true,
    "user_level_str": "专业版会员"
  }
}
```

---

## 二、钱包模块

### 2.1 获取钱包余额

```
GET /api/y/v1/wallet
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/y/v1/wallet' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "charge_balance": "65.3w",
    "gift_balance": "3.1w",
    "total_balance": "68.4w",
    "charge_balance_amount": 653633,
    "gift_balance_amount": 31000,
    "total_balance_amount": 684633
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `charge_balance` | string | 充值金币（可读格式） |
| `gift_balance` | string | 赠送银币（可读格式） |
| `total_balance` | string | 总余额（可读格式） |
| `charge_balance_amount` | int | 充值金币（精确数值） |
| `gift_balance_amount` | int | 赠送银币（精确数值） |
| `total_balance_amount` | int | 总余额（精确数值） |

> **币种说明**：金币(coin_type=2) = 充值获得；银币(coin_type=1) = 赠送获得。消耗优先使用即将过期的币（先进先出）。

---

### 2.2 获取 BZ 币明细列表

```
GET /api/y/v1/coins?expire_days={days}&current={page}&page_size={size}&coin_type={type}
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `expire_days` | int | 是 | 筛选过期天数范围：1/7/15/31/372(1年内) |
| `current` | int | 是 | 页码，从 1 开始 |
| `page_size` | int | 是 | 每页条数 |
| `coin_type` | int | 否 | 币种筛选：1=银币, 2=金币。不传=全部 |

**curl**：

```bash
# 查看所有币（1年内过期）
curl -s 'https://bizyair.cn/api/y/v1/coins?expire_days=372&current=1&page_size=10' \
  -H "Authorization: Bearer $TOKEN"

# 仅查看金币
curl -s 'https://bizyair.cn/api/y/v1/coins?expire_days=372&current=1&page_size=10&coin_type=2' \
  -H "Authorization: Bearer $TOKEN"

# 仅查看银币，7天内过期
curl -s 'https://bizyair.cn/api/y/v1/coins?expire_days=7&current=1&page_size=10&coin_type=1' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "list": [
      {
        "coin_type": 2,
        "amount": "1.5w",
        "created_at": "2026-05-26 11:37:39",
        "expired_at": "2026-06-25"
      }
    ],
    "total": 67,
    "current": 1,
    "page_size": 10,
    "total_charge_coins": "65.3w",
    "total_gift_coins": "3.1w",
    "total_charge_coins_amount": 653633,
    "total_gift_coins_amount": 31000
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `list[].coin_type` | int | 1=银币(赠送), 2=金币(充值) |
| `list[].amount` | string | 币量（可读格式） |
| `list[].created_at` | string | 获得时间 |
| `list[].expired_at` | string | 过期时间 |
| `total` | int | 总条数 |
| `total_charge_coins_amount` | int | 筛选范围内金币总额 |
| `total_gift_coins_amount` | int | 筛选范围内银币总额 |

---

## 三、兑换码模块

### 3.1 查询并兑换兑换码

```
GET /api/x/v1/share/{code}
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `code` | string | 兑换码（6-10位字母数字组合，区分大小写） |

> **注意**：此接口同时完成查询和兑换。已登录用户调用即执行兑换操作。

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/share/NJqr9Kyp' \
  -H "Authorization: Bearer $TOKEN"
```

**成功响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "data": {
      "biz_id": "3",
      "product_name": "20000 BZ币",
      "type": "redeem_product"
    },
    "type": "redeem_product"
  }
}
```

**兑换物品类型**：

| type | biz_item_type | 说明 |
|------|---------------|------|
| `redeem_product` | `coin` | BZ 币 |
| `redeem_product` | `user_level` | 会员等级（如专业版） |

---

### 3.2 获取兑换记录

```
GET /api/x/v1/share/redemptions?current={page}&page_size={size}
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `current` | int | 是 | 页码，从 1 开始 |
| `page_size` | int | 是 | 每页条数 |

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/share/redemptions?current=1&page_size=10' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "redemptions": [
      {
        "redeemed_at": "2026-05-26 13:47:56",
        "biz_item_name": "20000 BZ币",
        "biz_item_type": "coin",
        "code": "NJqr9Kyp",
        "issued_at": "可选字段，仅部分记录有"
      }
    ],
    "total": 5,
    "current": 1,
    "page_size": 10
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `redeemed_at` | string | 兑换时间 |
| `biz_item_name` | string | 物品名称 |
| `biz_item_type` | string | `coin`=BZ币, `user_level`=会员等级 |
| `code` | string | 兑换码 |
| `issued_at` | string | 可选，发行时间 |

---

## 四、通知/动态模块

### 4.1 获取未读动态数

```
GET /api/x/v1/dynamics/unread
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/dynamics/unread' \
  -H "Authorization: Bearer $TOKEN"
```

---

### 4.2 获取通知列表

```
GET /api/x/v1/notifications?page_size={size}
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page_size` | int | 是 | 每页条数 |

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/notifications?page_size=10' \
  -H "Authorization: Bearer $TOKEN"
```

---

### 4.3 获取未读通知数

```
GET /api/x/v1/notifications/unread_count
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/notifications/unread_count' \
  -H "Authorization: Bearer $TOKEN"
```

---

## 五、其他接口

### 5.1 获取 Banner 展示

```
GET /api/x/v1/banner_display?display_area={area}
```

**参数**：

| display_area | 说明 |
|--------------|------|
| `HEADER` | 顶部 |
| `AFTER_LOGIN_POP` | 登录后弹窗 |
| `HOME_TOP` | 首页顶部 |

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/banner_display?display_area=HEADER' \
  -H "Authorization: Bearer $TOKEN"
```

---

### 5.2 获取开源项目列表

```
GET /api/x/v1/open_source_project?current={page}&page_size={size}
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/open_source_project?current=1&page_size=8' \
  -H "Authorization: Bearer $TOKEN"
```

---

## 六、模型服务模块

### 6.1 获取模型标签列表

```
GET /api/x/v1/modelzoo/tags
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/tags' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "tags": [
      "通用图片B", "通用图片O", "通用视频X", "Happy Horse",
      "通用图片F", "通用视觉G", "通用对话G", "Seedream",
      "通用视频V", "万相图片", "万相视频", "海螺",
      "Seedance", "DreamActor", "可灵", "Vidu",
      "SkyReels", "Mureka", "自部署开源模型", "最近上新"
    ]
  }
}
```

---

### 6.2 获取模型分类列表

```
GET /api/x/v1/modelzoo/categories?tags={tag}
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tags` | string | 否 | 按标签筛选分类，不传返回全部分类 |

**curl**：

```bash
# 获取全部分类
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/categories' \
  -H "Authorization: Bearer $TOKEN"

# 按标签筛选
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/categories?tags=Happy+Horse' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "list": [
      { "category": "Text to Image", "api_count": 14 },
      { "category": "Image to Image", "api_count": 13 },
      { "category": "Text to Video", "api_count": 21 },
      { "category": "Image to Video", "api_count": 15 },
      { "category": "FLF to Video", "api_count": 14 },
      { "category": "Reference to Video", "api_count": 8 },
      { "category": "Video Edit", "api_count": 3 },
      { "category": "Video Extend", "api_count": 1 },
      { "category": "Large Language Models", "api_count": 3 },
      { "category": "Text to Speech", "api_count": 1 },
      { "category": "Vision", "api_count": 4 }
    ]
  }
}
```

**category 枚举值**：

| category | 说明 |
|----------|------|
| `Text to Image` | 文生图 |
| `Image to Image` | 图生图 |
| `Text to Video` | 文生视频 |
| `Image to Video` | 图生视频 |
| `FLF to Video` | 首尾帧生视频 |
| `Reference to Video` | 参考生视频 |
| `Video Edit` | 视频编辑 |
| `Video Extend` | 视频延长 |
| `Large Language Models` | 大语言模型 |
| `Text to Speech` | 文生语音 |
| `Vision` | 视觉理解 |

---

### 6.3 获取模型列表

```
POST /api/x/v1/modelzoo/list?page_size={size}&sort={sort}&current={page}
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page_size` | int | 是 | 每页条数 |
| `sort` | string | 是 | 排序方式：`recently_add`(最近添加) |
| `current` | int | 是 | 页码，从 1 开始 |

**请求体 (JSON)**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tags` | string[] | 否 | 按标签筛选，如 `["Happy Horse"]` |
| `categories` | string[] | 否 | 按分类筛选，如 `["Text to Image"]` |

**curl**：

```bash
# 无筛选，获取全部
curl -s -X POST 'https://bizyair.cn/api/x/v1/modelzoo/list?page_size=16&sort=recently_add&current=1' \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'

# 按标签筛选
curl -s -X POST 'https://bizyair.cn/api/x/v1/modelzoo/list?page_size=16&sort=recently_add&current=1' \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tags":["Happy Horse"]}'

# 按分类筛选
curl -s -X POST 'https://bizyair.cn/api/x/v1/modelzoo/list?page_size=16&sort=recently_add&current=1' \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"categories":["Text to Image"]}'
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "list": [
      {
        "id": 1083,
        "display_name": "HappyHorse-1.0-图生视频-官方版-基于首帧",
        "description": "HappyHorse 1.0 官方版能将静态图像转化为...",
        "category": "Image to Video",
        "model_name": "happyhorse-1-0-official",
        "endpoint": "happyhorse-1-0-official/image-to-video",
        "tags": ["Happy Horse", "最近上新"],
        "status": "normal",
        "new_tag": true,
        "created_at": "2026-05-14 21:25:23",
        "updated_at": "2026-05-14 14:19:09",
        "owner": "Third",
        "manufacturer": "阿里",
        "icon_url": "https://storage.bizyair.cn/icons/models/happyHorse_1.png",
        "background_url": "https://storage.bizyair.cn/icons/model_bg/image3.webp",
        "sort": 50
      }
    ],
    "total": 97,
    "current": 1,
    "page_size": 16
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 模型服务唯一 ID |
| `display_name` | string | 展示名称 |
| `description` | string | 模型描述 |
| `category` | string | 分类（见 6.2） |
| `model_name` | string | 模型名称（同一模型可有多个 endpoint） |
| `endpoint` | string | API 端点路径，格式：`{model_name}/{action}` |
| `tags` | string[] | 标签列表 |
| `status` | string | 状态：`normal` |
| `new_tag` | bool | 是否为新上线 |
| `owner` | string | 归属：`Third`(第三方), `Self`(自部署) |
| `manufacturer` | string | 厂商 |

---

### 6.4 获取模型服务详情

```
GET /api/x/v1/modelzoo/detail/{model_name}/{action}
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `model_name` | string | 模型名称，如 `z-image-turbo` |
| `action` | string | 动作，如 `text-to-image` |

> 路径格式对应列表接口的 `endpoint` 字段，将 `/` 分割为两部分。

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/detail/z-image-turbo/text-to-image' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "id": 1017,
    "display_name": "通义Z-Image.Turbo-文生图-官方版",
    "description": "阿里推出的Z-Image-Turbo极速文生图像模型...",
    "category": "Text to Image",
    "model_name": "z-image-turbo",
    "endpoint": "z-image-turbo/text-to-image",
    "tags": ["自部署开源模型"],
    "status": "normal",
    "created_at": "2026-05-06 15:00:16",
    "updated_at": "2026-05-14 20:40:47",
    "owner": "Self",
    "manufacturer": "硅基流动",
    "icon_url": "https://storage.bizyair.cn/icons/models/siliconflow.png",
    "background_url": "https://storage.bizyair.cn/icons/model_bg/image9.webp",
    "related_categories": [
      {
        "id": 1017,
        "category": "Text to Image",
        "endpoint": "z-image-turbo/text-to-image"
      }
    ],
    "input_params": [
      {
        "field_name": "prompt",
        "field_type": "customtext",
        "field_label": "提示词",
        "field_value": "默认值...",
        "variable_type": "string",
        "variable_name": "prompt",
        "required": true
      },
      {
        "field_name": "negative_prompt",
        "field_type": "customtext",
        "field_label": "负向提示词",
        "variable_type": "string",
        "variable_name": "negative_prompt"
      },
      {
        "field_name": "batch_size",
        "field_type": "slides",
        "field_label": "生成数量",
        "field_value": 1,
        "variable_type": "number",
        "variable_name": "batch_size",
        "required": true,
        "billing_dim": true,
        "field_options": { "max": 4, "min": 1, "step": 1 }
      },
      {
        "field_name": "seed",
        "field_type": "seed",
        "field_label": "种子",
        "field_value": -1,
        "variable_type": "number",
        "variable_name": "seed",
        "field_options": { "max": 2147483647, "min": 1, "step": 5 }
      },
      {
        "field_name": "height",
        "field_type": "number",
        "field_label": "图片高度",
        "field_value": 1024,
        "variable_type": "number",
        "variable_name": "height",
        "field_options": { "max": 2048, "min": 256, "step": 1 }
      },
      {
        "field_name": "width",
        "field_type": "number",
        "field_label": "图像宽度",
        "field_value": 1024,
        "variable_type": "number",
        "variable_name": "width",
        "field_options": { "max": 2048, "min": 256, "step": 1 }
      }
    ],
    "outputs_example": {
      "images": [
        "https://storage.bizyair.cn/outputs/xxx.png"
      ]
    }
  }
}
```

**input_params 字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `field_name` | string | 字段名 |
| `field_type` | string | UI 控件类型：`customtext`/`slides`/`seed`/`number` |
| `field_label` | string | 中文标签 |
| `field_value` | any | 默认值 |
| `variable_type` | string | 变量类型：`string`/`number` |
| `variable_name` | string | 变量名（API 调用时使用） |
| `required` | bool | 是否必填 |
| `billing_dim` | bool | 是否计费维度（如 batch_size 影响价格） |
| `field_options` | object | 数值范围 `{min, max, step}` |

---

### 6.5 预估调用价格

```
POST /api/x/v1/modelzoo/indicative_price/{model_name}/{action}
```

**请求体**：与模型 `input_params` 对应的参数键值对。

**curl**：

```bash
curl -s -X POST 'https://bizyair.cn/api/x/v1/modelzoo/indicative_price/z-image-turbo/text-to-image' \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A red car",
    "negative_prompt": "",
    "batch_size": 1,
    "seed": -1,
    "height": 1024,
    "width": 1024
  }'
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "coin_type": 2,
    "result": "5"
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `coin_type` | int | 币种：1=银币, 2=金币 |
| `result` | string | 预估价格（币数） |

---

### 6.6 获取模型价格表

```
GET /api/x/v1/modelzoo/price_table/{model_name}/{action}
```

**curl**：

```bash
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/price_table/z-image-turbo/text-to-image' \
  -H "Authorization: Bearer $TOKEN"
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "price_table": {
      "columns": [
        { "variable_name": "batch_size", "field_label": "生成数量" },
        { "variable_name": "price", "field_label": "价格 (¥)" }
      ],
      "simple_price_text": "总像素数<=1024x1024，5金币/张；总像素数>1024x1024，10金币/张；"
    },
    "benefit": {
      "rpd": 5000,
      "rph": 300,
      "rpm": -1
    }
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `price_table.columns` | array | 价格表列定义 |
| `price_table.simple_price_text` | string | 简易价格描述文本 |
| `benefit.rpd` | int | 每日请求限制 (Requests Per Day) |
| `benefit.rph` | int | 每小时请求限制 (Requests Per Hour) |
| `benefit.rpm` | int | 每分钟请求限制 (Requests Per Minute)，-1=无限制 |

---

## 七、模型推理任务模块

### 7.1 提交推理任务

```
POST /api/x/v1/modelzoo/tasks/{model_name}/{action}
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `model_name` | string | 模型名称，如 `z-image-turbo` |
| `action` | string | 动作，如 `text-to-image` |

**请求体**：与模型详情的 `input_params` 对应，键为 `variable_name`。

**curl**：

```bash
curl -s -X POST 'https://bizyair.cn/api/x/v1/modelzoo/tasks/z-image-turbo/text-to-image' \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A beautiful sunset over the ocean with golden light",
    "negative_prompt": "",
    "batch_size": 1,
    "seed": -1,
    "height": 1024,
    "width": 1024
  }'
```

**响应**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "request_id": "60673dd2-3ade-4363-8727-8465e5961621"
  }
}
```

> 返回 `request_id` 后需轮询 7.2 接口获取结果。

---

### 7.2 查询任务状态

```
GET /api/x/v1/modelzoo/tasks/{request_id}
```

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `request_id` | string | 提交任务时返回的请求 ID |

**curl**：

```bash
# 单次查询
curl -s 'https://bizyair.cn/api/x/v1/modelzoo/tasks/60673dd2-3ade-4363-8727-8465e5961621' \
  -H "Authorization: Bearer $TOKEN"

# 轮询直到完成（bash 示例）
REQUEST_ID="60673dd2-3ade-4363-8727-8465e5961621"
while true; do
  RESULT=$(curl -s "https://bizyair.cn/api/x/v1/modelzoo/tasks/$REQUEST_ID" \
    -H "Authorization: Bearer $TOKEN")
  STATUS=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['status'])")
  echo "Status: $STATUS"
  if [ "$STATUS" != "Running" ]; then
    echo "$RESULT" | python3 -m json.tool
    break
  fi
  sleep 2
done
```

**响应（运行中）**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "request_id": "60673dd2-3ade-4363-8727-8465e5961621",
    "status": "Running",
    "created_at": "2026-05-26 14:02:16",
    "executed_at": "2026-05-26 14:02:16",
    "outputs": {},
    "cost_times": {}
  }
}
```

**响应（已完成）**：

```json
{
  "code": 20000,
  "message": "Ok",
  "status": true,
  "data": {
    "request_id": "60673dd2-3ade-4363-8727-8465e5961621",
    "status": "Success",
    "created_at": "2026-05-26 14:02:16",
    "executed_at": "2026-05-26 14:02:16",
    "ended_at": "2026-05-26 14:02:21",
    "outputs": {
      "images": [
        "https://bizyair-prod.oss-cn-shanghai.aliyuncs.com/outputs/xxx.png"
      ]
    },
    "cost_times": {
      "total_cost_time": 5112,
      "inference_duration": 5068
    }
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `request_id` | string | 任务唯一 ID (UUID) |
| `status` | string | 任务状态：`Running`/`Success`/`Failed` |
| `created_at` | string | 创建时间 |
| `executed_at` | string | 开始执行时间 |
| `ended_at` | string | 完成时间（仅 Success 时有） |
| `outputs` | object | 输出结果，key 按类型：`images`(图片)/`videos`(视频) |
| `cost_times.total_cost_time` | int | 总耗时（毫秒） |
| `cost_times.inference_duration` | int | 推理耗时（毫秒） |

**轮询策略**：前端观察为每 1-2 秒轮询一次，直到 `status` 变为 `Success` 或 `Failed`。

---

## 变更记录

| 日期 | 操作 |
|------|------|
| 2026-05-26 | 为所有接口添加 curl 示例命令 |
| 2026-05-26 | 新增模型推理任务模块（7.1-7.2） |
| 2026-05-26 | 新增模型服务模块（6.1-6.6） |
| 2026-05-26 | 初始版本：用户、钱包、兑换码、通知/动态模块 |

# Cấu trúc dữ liệu cấu hình người dùng lưu bằng Redis

#### I. Cấu hình

##### 1. Cấu trúc hget của cấu hình toàn cục

```text
xiaozhi:global:config
```

##### 2. Cấu hình người dùng có thể ghi đè cấu hình trong file, dùng cấu trúc hget

```json
xiaozhi:userconfig:{deviceid}
{
  "llm": {
    "provider": "deepseek" // Tương ứng với key trong cấu hình llm.
  },
  "tts": {
    "provider": "cosyvoice" // Tương ứng với key trong cấu hình tts.
  }
}
```

#### II. Prompt

##### 1. get/set prompt hệ thống

```text
xiaozhi:llm:system:{deviceid}
```

##### 2. Cấu trúc sorted set ghi prompt của session chat

```text
xiaozhi:llm:{deviceid}
```

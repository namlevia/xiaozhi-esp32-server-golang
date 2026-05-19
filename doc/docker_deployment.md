# Hỗ trợ build Docker local

Đã thêm file `docker-compose.local.yml`, hỗ trợ build local và triển khai đa kiến trúc.

## File mới

- `docker/docker-composer/docker-compose.local.yml` - file cấu hình build local

## Cách build

### Biên dịch mặc định (AMD64)

```bash
cd docker/docker-composer
docker-compose -f docker-compose.local.yml up --build
```

### Biên dịch ARM64 (Apple Silicon)

```bash
cd docker/docker-composer
TARGETARCH=arm64 docker-compose -f docker-compose.local.yml up --build
```

## Cách chạy

Sau khi build xong, service sẽ tự khởi động, bao gồm:

- Server chính (cổng 8989)
- Backend quản trị (cổng 8081)
- Giao diện frontend (cổng 8080)
- Database MySQL (cổng 23306)

Truy cập http://<IP hoặc domain server>:8080 để xem giao diện frontend.

## 🏗️ Hỗ trợ đa kiến trúc

### Tự động phát hiện kiến trúc (khuyến nghị)

`docker-compose.local.yml` hỗ trợ tự động phát hiện kiến trúc hệ thống hiện tại:

```bash
# Tự động phát hiện kiến trúc và build (hành vi mặc định)
docker-compose -f docker-compose.local.yml up --build
```

### Chỉ định kiến trúc thủ công

Nếu cần build cho kiến trúc cụ thể:

```bash
# Build cho kiến trúc ARM64
TARGETARCH=arm64 docker-compose -f docker-compose.local.yml up --build

# Build cho kiến trúc AMD64
TARGETARCH=amd64 docker-compose -f docker-compose.local.yml up --build
```

### Kiến trúc được hỗ trợ

- **AMD64/x86_64**: CPU Intel/AMD (mặc định)
- **ARM64**: Apple Silicon (M1/M2), server ARM

## 📁 Mô tả file cấu hình

### docker-compose.yml

Dùng image chính thức đã build sẵn, phù hợp môi trường production:

```yaml
services:
  mysql:
    image: docker.jsdelivr.fyi/mysql:8.0
  main-server:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_golang:0.1
  backend:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_backend:0.1
  frontend:
    image: docker.jsdelivr.fyi/hackers365/xiaozhi_frontend:0.1
```

### docker-compose.local.yml

Bản build local, hỗ trợ sửa code và đa kiến trúc:

```yaml
services:
  main-server:
    build:
      context: ../..
      dockerfile: docker/Dockerfile.main
      args:
        TARGETARCH: ${TARGETARCH:-amd64}
```

## 🔧 Cấu hình biến môi trường

### Liên quan kiến trúc

| Tên biến | Mặc định | Mô tả |
|-------|-------|------|
| `TARGETARCH` | `amd64` | Kiến trúc mục tiêu (`amd64`/`arm64`) |

## 🛠️ Thao tác thường gặp

### Xem trạng thái service

```bash
# Xem trạng thái toàn bộ service
docker-compose ps

# Xem log service
docker-compose logs -f main-server
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Dừng và restart service

```bash
# Dừng toàn bộ service
docker-compose down

# Restart service cụ thể
docker-compose restart main-server

# Build lại và khởi động
docker-compose up --build
```

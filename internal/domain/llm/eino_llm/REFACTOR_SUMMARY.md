# Tổng kết refactor ResponseWithFunctions

## Mục tiêu

Refactor `ResponseWithFunctions` để gọi trực tiếp `EinoResponseWithTools`, loại bỏ code trùng lặp và tăng khả năng tái sử dụng.

## Trước khi refactor

`ResponseWithFunctions` tự xử lý bind tool, streaming, non-streaming và chuyển đổi format response nên bị trùng logic với `EinoResponseWithTools`.

## Sau khi refactor

`ResponseWithFunctions` đóng vai trò adapter: gọi `EinoResponseWithTools`, rồi chuyển response native của Eino về format cũ để giữ tương thích với caller hiện tại.

## Lợi ích

- Giảm code trùng lặp.
- Tập trung logic tool call ở một nơi.
- Dễ bảo trì streaming/non-streaming hơn.
- Giữ tương thích API hiện tại.

# Postman коллекция: MWork + PhotoStudio (полный набор)

## Импорт в Postman
1. Откройте Postman.
2. Import → File → выберите:
   - `postman/MWork_PhotoStudio.postman_collection.json`
   - `postman/MWork_PhotoStudio.postman_environment.json`
3. Выберите окружение **MWork + PhotoStudio (Local)**.

## Порядок и важность
Коллекция отсортирована сверху вниз по важности для фронтенд-разработки:
1) Health/Auth → 2) Profiles → 3) Castings/Responses → 4) Photos/Files → 5) Chat/Notifications → 6) Subscriptions/Payments/Dashboard → 7) Promotions/Reviews/FAQ → 8) Moderation → 9) PhotoStudio → 10) Leads/Admin/Webhooks.

## Переменные окружения
Заполните значения перед запуском:
- `mwork_base_url` (например, `http://localhost:8080`)
- `photostudio_base_url` (например, `http://localhost:8090`)
- `client_email` / `client_password`
- `photostudio_token` (токен для внутренних запросов к PhotoStudio)
- `studio_id` / `room_id` (нужны для создания бронирования)
- `mwork_user_id` (UUID пользователя для внутренних запросов к PhotoStudio)
- остальные `*_id` и `*_token` переменные — по необходимости для конкретных запросов

`mwork_token`, `mwork_refresh_token`, `booking_id`, `start_time`, `end_time` заполняются автоматически тестами/скриптами.

## Запуск через newman
```bash
newman run postman/MWork_PhotoStudio.postman_collection.json \
  -e postman/MWork_PhotoStudio.postman_environment.json
```

## Примечания
- Коллекция использует **реальные эндпоинты**, найденные в коде этого репозитория.
- В PhotoStudio нет явных `/health` или `/auth` роутов в этом репозитории, поэтому раздел содержит доступные внутренние endpoints.

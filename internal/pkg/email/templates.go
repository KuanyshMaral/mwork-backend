package email

// Email templates in HTML format

// BaseTemplate is the base layout for all emails
const BaseTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #0f0f0f;
            color: #ffffff;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        .card {
            background: #1a1a1a;
            border-radius: 12px;
            padding: 32px;
            border: 1px solid #2a2a2a;
        }
        .logo {
            text-align: center;
            margin-bottom: 24px;
        }
        .logo h1 {
            font-size: 28px;
            background: linear-gradient(135deg, #a855f7 0%, #6366f1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin: 0;
        }
        h2 {
            color: #ffffff;
            font-size: 24px;
            margin: 0 0 16px;
        }
        p {
            color: #888888;
            font-size: 16px;
            line-height: 1.6;
            margin: 0 0 16px;
        }
        .btn {
            display: inline-block;
            background: linear-gradient(135deg, #a855f7 0%, #6366f1 100%);
            color: #ffffff !important;
            text-decoration: none;
            padding: 14px 28px;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            margin: 16px 0;
        }
        .footer {
            text-align: center;
            margin-top: 32px;
            color: #666666;
            font-size: 12px;
        }
        .highlight {
            color: #a855f7;
            font-weight: 600;
        }
        .info-box {
            background: #252525;
            border-radius: 8px;
            padding: 16px;
            margin: 16px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <div class="logo">
                <h1>MWork</h1>
            </div>
            {{.Content}}
        </div>
        <div class="footer">
            <p>© 2026 MWork. Все права защищены.</p>
            <p>Вы получили это письмо, потому что зарегистрированы на mwork.kz</p>
        </div>
    </div>
</body>
</html>
`

// ResponseAcceptedTemplate - notification when response is accepted
const ResponseAcceptedTemplate = `
<h2>🎉 Вас приняли на кастинг!</h2>
<p>Поздравляем, <span class="highlight">{{.ModelName}}</span>!</p>
<p>Ваша заявка на кастинг <strong>"{{.CastingTitle}}"</strong> была принята.</p>
<div class="info-box">
    <p><strong>Работодатель:</strong> {{.EmployerName}}</p>
    {{if .CastingDate}}<p><strong>Дата:</strong> {{.CastingDate}}</p>{{end}}
</div>
<p>Свяжитесь с работодателем для уточнения деталей.</p>
<a href="{{.CastingURL}}" class="btn">Подробности кастинга</a>
`

// ResponseRejectedTemplate - notification when response is rejected
const ResponseRejectedTemplate = `
<h2>Заявка отклонена</h2>
<p>К сожалению, ваша заявка на кастинг <strong>"{{.CastingTitle}}"</strong> была отклонена.</p>
<p>Не расстраивайтесь! На платформе много других интересных кастингов.</p>
<a href="{{.CastingsURL}}" class="btn">Смотреть кастинги</a>
`

// NewResponseTemplate - notification for employer about new response
const NewResponseTemplate = `
<h2>📩 Новый отклик на кастинг</h2>
<p>На ваш кастинг <strong>"{{.CastingTitle}}"</strong> откликнулась модель.</p>
<div class="info-box">
    <p><strong>Модель:</strong> {{.ModelName}}</p>
    {{if .ModelAge}}<p><strong>Возраст:</strong> {{.ModelAge}} лет</p>{{end}}
    {{if .ModelCity}}<p><strong>Город:</strong> {{.ModelCity}}</p>{{end}}
</div>
<a href="{{.ResponseURL}}" class="btn">Посмотреть заявку</a>
`

// NewMessageTemplate - notification about new chat message
const NewMessageTemplate = `
<h2>💬 Новое сообщение</h2>
<p>У вас новое сообщение от <span class="highlight">{{.SenderName}}</span>:</p>
<div class="info-box">
    <p>"{{.MessagePreview}}"</p>
</div>
<a href="{{.ChatURL}}" class="btn">Открыть чат</a>
`

// CastingExpiringTemplate - notification for employer about expiring casting
const CastingExpiringTemplate = `
<h2>⏰ Кастинг скоро завершится</h2>
<p>Ваш кастинг <strong>"{{.CastingTitle}}"</strong> завершится через {{.DaysLeft}} дней.</p>
<p>Всего откликов: <span class="highlight">{{.ResponseCount}}</span></p>
<a href="{{.CastingURL}}" class="btn">Управление кастингом</a>
`

// WelcomeTemplate - welcome email for new users
const WelcomeTemplate = `
<h2>Добро пожаловать в MWork! 🎉</h2>
<p>Привет, <span class="highlight">{{.UserName}}</span>!</p>
<p>Вы успешно зарегистрировались на платформе MWork — крупнейшей площадке для моделей и работодателей в Казахстане.</p>
{{if eq .Role "model"}}
<p>Что дальше?</p>
<ul>
    <li>Заполните профиль и добавьте фотографии</li>
    <li>Просматривайте кастинги и откликайтесь</li>
    <li>Подключите Pro-подписку для больше возможностей</li>
</ul>
{{else}}
<p>Что дальше?</p>
<ul>
    <li>Создайте свой первый кастинг</li>
    <li>Получайте отклики от моделей</li>
    <li>Выбирайте лучших кандидатов</li>
</ul>
{{end}}
<a href="{{.DashboardURL}}" class="btn">Перейти в личный кабинет</a>
`

// LeadApprovedTemplate - notification when company lead is approved
const LeadApprovedTemplate = `
<h2>✅ Ваша заявка одобрена!</h2>
<p>Здравствуйте, <span class="highlight">{{.ContactName}}</span>!</p>
<p>Ваша заявка от компании <strong>{{.CompanyName}}</strong> была рассмотрена и одобрена.</p>
<p>Мы создали для вас аккаунт работодателя на платформе MWork.</p>
<div class="info-box">
    <p><strong>Email:</strong> {{.Email}}</p>
    <p><strong>Временный пароль:</strong> {{.TempPassword}}</p>
</div>
<p>Рекомендуем сменить пароль после первого входа.</p>
<a href="{{.LoginURL}}" class="btn">Войти в аккаунт</a>
`

// LeadRejectedTemplate - notification when company lead is rejected
const LeadRejectedTemplate = `
<h2>Заявка рассмотрена</h2>
<p>Здравствуйте, <span class="highlight">{{.ContactName}}</span>!</p>
<p>К сожалению, ваша заявка от компании <strong>{{.CompanyName}}</strong> была отклонена.</p>
{{if .Reason}}
<div class="info-box">
    <p><strong>Причина:</strong> {{.Reason}}</p>
</div>
{{end}}
<p>Если у вас есть вопросы, свяжитесь с нами по адресу support@mwork.kz</p>
`

// DigestTemplate - weekly/daily digest email
const DigestTemplate = `
<h2>📊 Ваша сводка за неделю</h2>
<p>Привет, <span class="highlight">{{.UserName}}</span>! Вот что произошло:</p>
<div class="info-box">
    {{if .NewResponses}}<p>📩 Новых откликов: <strong>{{.NewResponses}}</strong></p>{{end}}
    {{if .NewMessages}}<p>💬 Новых сообщений: <strong>{{.NewMessages}}</strong></p>{{end}}
    {{if .ProfileViews}}<p>👁 Просмотров профиля: <strong>{{.ProfileViews}}</strong></p>{{end}}
    {{if .NewCastings}}<p>🎬 Новых кастингов по вашим критериям: <strong>{{.NewCastings}}</strong></p>{{end}}
</div>
<a href="{{.DashboardURL}}" class="btn">Открыть личный кабинет</a>
`

// VerificationTemplate - email verification code
const VerificationTemplate = `
<h2>📧 Подтвердите ваш email</h2>
<p>Здравствуйте, <span class="highlight">{{.UserName}}</span>!</p>
<p>Для подтверждения вашего email-адреса введите следующий код:</p>
<div class="info-box" style="text-align: center;">
    <p style="font-size: 32px; font-weight: 700; letter-spacing: 8px; color: #a855f7; margin: 0;">{{.Code}}</p>
</div>
<p>Код действителен в течение 15 минут.</p>
<p style="color: #666;">Если вы не регистрировались на MWork, проигнорируйте это письмо.</p>
`

// PasswordResetTemplate - password reset link
const PasswordResetTemplate = `
<h2>🔐 Сброс пароля</h2>
<p>Здравствуйте, <span class="highlight">{{.UserName}}</span>!</p>
<p>Вы запросили сброс пароля для вашего аккаунта на MWork.</p>
<p>Нажмите на кнопку ниже, чтобы установить новый пароль:</p>
<a href="{{.ResetURL}}" class="btn">Сбросить пароль</a>
<p style="color: #666; margin-top: 20px;">Ссылка действительна в течение 1 часа.</p>
<p style="color: #666;">Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>
`

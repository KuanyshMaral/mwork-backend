package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/subscription"
	"github.com/mwork/mwork-api/internal/pkg/robokassa"
	"github.com/rs/zerolog/log"
)

// Service handles payment business logic
type Service struct {
	repo            Repository
	subSvc          *subscription.Service
	creditSvc       credit.Service // ✅ FIXED: Using credit.Service interface
	robokassaConfig RobokassaConfig
}

// RobokassaConfig содержит настройки интеграции с платежной системой Robokassa
type RobokassaConfig struct {
	MerchantLogin string                  // Идентификатор магазина в Robokassa
	Password1     string                  // Пароль #1 для формирования подписи
	Password2     string                  // Пароль #2 для проверки результата оплаты
	TestPassword1 string                  // Тестовый пароль #1
	TestPassword2 string                  // Тестовый пароль #2
	IsTest        bool                    // Режим тестирования
	HashAlgo      robokassa.HashAlgorithm // Алгоритм хеширования (MD5, SHA256, SHA512)
	PaymentURL    string                  // URL платежной формы Robokassa
	ResultURL     string                  // URL для получения результата оплаты
	SuccessURL    string                  // URL для редиректа после успешной оплаты
	FailURL       string                  // URL для редиректа после неудачной оплаты
}

// NewService создает новый экземпляр сервиса платежей
func NewService(repo Repository, subSvc *subscription.Service) *Service {
	return &Service{
		repo:   repo,
		subSvc: subSvc,
	}
}

// SetRobokassaConfig устанавливает конфигурацию Robokassa
func (s *Service) SetRobokassaConfig(cfg RobokassaConfig) {
	s.robokassaConfig = cfg
}

// InitRobokassaPaymentRequest содержит параметры для инициализации платежа через Robokassa
type InitRobokassaPaymentRequest struct {
	UserID         uuid.UUID // ID пользователя
	SubscriptionID uuid.UUID // ID подписки
	Amount         string    // Сумма платежа (строка для точности)
	Description    string    // Описание платежа
}

// InitRobokassaPaymentResponse содержит данные созданного платежа
type InitRobokassaPaymentResponse struct {
	PaymentID  uuid.UUID `json:"payment_id"`  // ID платежа в системе
	InvID      int64     `json:"inv_id"`      // ID инвойса в Robokassa
	PaymentURL string    `json:"payment_url"` // URL для оплаты
	Status     string    `json:"status"`      // Статус платежа
}

// InitRobokassaPayment инициирует новый платеж через Robokassa.
// Создает запись о платеже в БД, генерирует подпись и возвращает URL для оплаты.
//
// Процесс:
// 1. Генерирует уникальный InvID на основе timestamp
// 2. Формирует подпись запроса согласно документации Robokassa
// 3. Создает запись о платеже со статусом pending
// 4. Возвращает URL платежной формы
//
// Возвращаемые ошибки:
//   - invalid amount: неверный формат суммы
//   - signing error: ошибка при генерации подписи
//   - database error: ошибка при создании записи в БД
func (s *Service) InitRobokassaPayment(ctx context.Context, req InitRobokassaPaymentRequest) (*InitRobokassaPaymentResponse, error) {
	invID := time.Now().UnixNano()
	amountRat, ok := new(big.Rat).SetString(req.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	outSum := amountRat.FloatString(2)
	shp := map[string]string{"Shp_payment_id": invIDString(invID)}
	password1 := s.robokassaConfig.Password1
	if s.robokassaConfig.IsTest && s.robokassaConfig.TestPassword1 != "" {
		password1 = s.robokassaConfig.TestPassword1
	}

	base := robokassa.BuildStartSignatureBase(s.robokassaConfig.MerchantLogin, outSum, invIDString(invID), password1, nil, shp)
	signature, err := robokassa.Sign(base, s.robokassaConfig.HashAlgo)
	if err != nil {
		return nil, err
	}

	initPayload := map[string]string{
		"MerchantLogin":  s.robokassaConfig.MerchantLogin,
		"OutSum":         outSum,
		"InvId":          invIDString(invID),
		"Description":    req.Description,
		"SignatureValue": signature,
		"ResultURL":      s.robokassaConfig.ResultURL,
		"SuccessURL":     s.robokassaConfig.SuccessURL,
		"FailURL":        s.robokassaConfig.FailURL,
		"Shp_payment_id": invIDString(invID),
	}
	if s.robokassaConfig.IsTest {
		initPayload["IsTest"] = "1"
	}

	rawInit, _ := json.Marshal(initPayload)
	payment := &Payment{
		ID:             uuid.New(),
		UserID:         req.UserID,
		SubscriptionID: uuid.NullUUID{UUID: req.SubscriptionID, Valid: true},
		Amount:         ratToFloat64(amountRat),
		Currency:       "KZT",
		Status:         StatusPending,
		Provider:       sql.NullString{String: "robokassa", Valid: true},
		ExternalID:     sql.NullString{String: invIDString(invID), Valid: true},
		RobokassaInvID: sql.NullInt64{Int64: invID, Valid: true},
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		RawInitPayload: rawInit,
	}

	if err := s.repo.CreateRobokassaPending(ctx, payment); err != nil {
		return nil, err
	}

	paymentURL := s.robokassaConfig.PaymentURL + "?" + encodeQuery(initPayload)
	return &InitRobokassaPaymentResponse{PaymentID: payment.ID, InvID: invID, PaymentURL: paymentURL, Status: string(StatusPending)}, nil
}

// ProcessRobokassaResult обрабатывает callback от Robokassa (Result URL).
// Выполняет проверку подписи, валидацию суммы и активацию подписки.
//
// Процесс:
// 1. Проверяет подпись запроса
// 2. Находит платеж по InvID
// 3. Проверяет соответствие суммы
// 4. Обновляет статус платежа на completed
// 5. Активирует подписку (если это платеж за подписку)
// 6. Создает событие в журнале
//
// Идемпотентность: повторные вызовы для уже обработанного платежа не вызывают ошибку.
//
// Возвращаемые ошибки:
//   - invalid signature: неверная подпись
//   - payment not found: платеж не найден
//   - amount mismatch: сумма не совпадает с ожидаемой
func (s *Service) ProcessRobokassaResult(ctx context.Context, outSum, invID, signature string, shp map[string]string, rawPayload map[string]string) error {
	password2 := s.robokassaConfig.Password2
	if s.robokassaConfig.IsTest && s.robokassaConfig.TestPassword2 != "" {
		password2 = s.robokassaConfig.TestPassword2
	}

	base := robokassa.BuildResultSignatureBase(outSum, invID, password2, shp)
	expectedSignature, err := robokassa.Sign(base, s.robokassaConfig.HashAlgo)
	if err != nil {
		return err
	}
	if !robokassa.VerifySignature(expectedSignature, signature) {
		return fmt.Errorf("invalid signature")
	}

	invIDInt, err := strconv.ParseInt(invID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid inv_id: %w", err)
	}
	callbackAmount, err := robokassa.ParseAmount(outSum)
	if err != nil {
		return err
	}

	tx, err := s.repo.BeginTxx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	payment, err := s.repo.GetByRobokassaInvIDForUpdate(ctx, tx, invIDInt)
	if err != nil {
		return err
	}
	if payment == nil {
		return ErrPaymentNotFound
	}
	if payment.Status == StatusCompleted {
		return tx.Commit()
	}

	expectedAmount, err := robokassa.ParseAmount(fmt.Sprintf("%.2f", payment.Amount))
	if err != nil {
		return err
	}
	if !robokassa.AmountsEqual(expectedAmount, callbackAmount) {
		return fmt.Errorf("amount mismatch")
	}

	if err := s.repo.MarkRobokassaSucceeded(ctx, tx, payment.ID, rawPayload); err != nil {
		return err
	}

	if payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			return err
		}
	}

	rawEvent, _ := json.Marshal(rawPayload)
	if err := s.repo.CreatePaymentEvent(ctx, tx, payment.ID, "robokassa.result.succeeded", rawEvent); err != nil {
		return err
	}

	return tx.Commit()
}

func invIDString(invID int64) string { return strconv.FormatInt(invID, 10) }

func encodeQuery(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

func ratToFloat64(v *big.Rat) float64 {
	out, _ := v.Float64()
	return out
}

// SetCreditService устанавливает сервис кредитов (опционально, для избежания циклических зависимостей)
func (s *Service) SetCreditService(creditSvc credit.Service) { // ✅ FIXED: Using credit.Service
	s.creditSvc = creditSvc
}

// CreatePayment создает новый платеж для подписки.
// Платеж создается в статусе pending и требует подтверждения через ConfirmPayment.
//
// Параметры:
//   - userID: ID пользователя
//   - subscriptionID: ID подписки
//   - amount: сумма платежа
//   - provider: платежный провайдер (напр. "robokassa", "stripe")
func (s *Service) CreatePayment(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, provider Provider) (*Payment, error) {
	now := time.Now()
	payment := &Payment{
		ID:             uuid.New(),
		UserID:         userID,
		SubscriptionID: uuid.NullUUID{UUID: subscriptionID, Valid: true},
		Amount:         amount,
		Currency:       "KZT",
		Status:         StatusPending,
		Provider:       sql.NullString{String: string(provider), Valid: true},
		CreatedAt:      now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// CreateCreditPayment создает новый платеж для покупки кредитов.
// Поддерживаются только фиксированные пакеты: 5, 10, 25, 50 кредитов.
//
// Валидация:
//   - Пакет кредитов должен быть одним из: 5, 10, 25, 50
//   - Платеж создается в статусе pending
//
// Возвращаемые ошибки:
//   - invalid credit package: неподдерживаемый пакет кредитов
func (s *Service) CreateCreditPayment(ctx context.Context, userID uuid.UUID, creditAmount int, totalPrice float64, provider Provider) (*Payment, error) {
	// Validate credit amount - only allow specific packages (5, 10, 25, 50)
	validPackages := map[int]bool{
		5:  true,
		10: true,
		25: true,
		50: true,
	}

	if !validPackages[creditAmount] {
		return nil, fmt.Errorf("invalid credit package: must be one of 5, 10, 25, or 50")
	}

	now := time.Now()
	payment := &Payment{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    totalPrice,
		Currency:  "KZT",
		Status:    StatusPending,
		Provider:  sql.NullString{String: string(provider), Valid: true},
		CreatedAt: now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// ConfirmPayment подтверждает платеж и выполняет соответствующие действия.
//
// Действия в зависимости от типа платежа:
//   - Платеж за подписку: активирует подписку
//   - Платеж за кредиты: начисляет кредиты на баланс пользователя
//
// Идемпотентность:
// Метод идемпотентен - повторные вызовы для уже обработанного платежа не приводят к дублированию кредитов или подписок.
//
// Возвращаемые ошибки:
//   - ErrPaymentNotFound: платеж не найден
func (s *Service) ConfirmPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil || payment == nil {
		return ErrPaymentNotFound
	}

	// B4: IDEMPOTENCY - If already paid, don't process again
	if payment.IsPaid() {
		return nil // Already processed - no duplicate credits
	}

	// Update payment status
	if err := s.repo.UpdateStatus(ctx, paymentID, StatusCompleted); err != nil {
		return err
	}

	// Activate subscription if this is a subscription payment
	if payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			log.Error().Err(err).Msg("Failed to activate subscription after payment")
		}
	} else if s.creditSvc != nil {
		// B4: GRANT CREDITS FOR CREDIT PURCHASE
		creditAmount := s.determineCreditAmount(payment.Amount)
		if creditAmount > 0 {
			paymentIDStr := payment.ID.String()
			creditMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
				RelatedEntityType: "payment",
				RelatedEntityID:   payment.ID,
				Description:       fmt.Sprintf("Purchase via %s: payment %s", payment.Provider.String, payment.ID.String()),
				PaymentID:         &paymentIDStr,
			}

			// Grant credits - idempotent at payment service level
			err := s.creditSvc.Add(ctx, payment.UserID, creditAmount, credit.TransactionTypePurchase, creditMeta) // ✅ FIXED: Using credit.TransactionTypePurchase
			if err != nil {
				log.Error().Err(err).Str("payment_id", payment.ID.String()).Msg("Failed to grant credits after payment")
			}
		}
	}

	return nil
}

// determineCreditAmount определяет количество кредитов по сумме платежа.
// Валидирует соответствие суммы ожидаемым пакетам кредитов.
//
// Тарифы (KZT):
//   - 5 кредитов = 500 ₸
//   - 10 кредитов = 900 ₸
//   - 25 кредитов = 2000 ₸
//   - 50 кредитов = 3500 ₸
//
// Возвращает 0, если сумма не соответствует ни одному пакету.
func (s *Service) determineCreditAmount(amount float64) int {
	// Allowed credit packages with pricing (KZT):
	// 5 credits = 500 KZT
	// 10 credits = 900 KZT
	// 25 credits = 2000 KZT
	// 50 credits = 3500 KZT
	priceToCredits := map[float64]int{
		500.0:  5,
		900.0:  10,
		2000.0: 25,
		3500.0: 50,
	}

	if credits, ok := priceToCredits[amount]; ok {
		return credits
	}

	return 0
}

// FailPayment отмечает платеж как неудавшийся.
// Обновляет статус платежа на failed.
func (s *Service) FailPayment(ctx context.Context, paymentID uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, paymentID, StatusFailed)
}

// HandleWebhook обрабатывает webhook от платежного провайдера.
//
// Процесс:
// 1. Находит платеж по внешнему ID
// 2. Проверяет идемпотентность (пропускает уже обработанные)
// 3. В зависимости от статуса вызывает ConfirmPayment или FailPayment
//
// Поддерживаемые статусы:
//   - success, completed, paid → подтверждение платежа
//   - failed, cancelled, declined → отклонение платежа
//
// Идемпотентность обеспечивается на уровне ConfirmPayment.
//
// Возвращаемые ошибки:
//   - ErrPaymentNotFound: платеж не найден
func (s *Service) HandleWebhook(ctx context.Context, provider string, externalID string, status string) error {
	payment, err := s.repo.GetByExternalID(ctx, provider, externalID)
	if err != nil || payment == nil {
		log.Warn().Str("provider", provider).Str("external_id", externalID).Msg("Payment not found for webhook")
		return ErrPaymentNotFound
	}

	// B4: Idempotency check - skip if already processed
	if payment.IsPaid() {
		log.Info().Str("payment_id", payment.ID.String()).Msg("Payment already processed, skipping duplicate webhook")
		return nil
	}

	switch status {
	case "success", "completed", "paid":
		return s.ConfirmPayment(ctx, payment.ID)
	case "failed", "cancelled", "declined":
		return s.FailPayment(ctx, payment.ID)
	default:
		log.Warn().Str("status", status).Msg("Unknown payment status in webhook")
	}

	return nil
}

// GetPaymentHistory возвращает историю платежей пользователя с пагинацией.
//
// Параметры:
//   - userID: ID пользователя
//   - limit: максимальное количество записей
//   - offset: смещение для пагинации
func (s *Service) GetPaymentHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

// Errors
var (
	ErrPaymentNotFound = subscription.ErrPaymentFailed
)

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"CheeseHouse/internal/config"
	"CheeseHouse/internal/models"
)

// WhatsAppService maneja toda la comunicación con WhatsApp Business API
type WhatsAppService struct {
	config        *config.Config
	client        *http.Client
	accessToken   string
	phoneNumberID string
	apiURL        string
}

// NewWhatsAppService crea una nueva instancia del servicio de WhatsApp
func NewWhatsAppService(cfg *config.Config) *WhatsAppService {
	return &WhatsAppService{
		config:        cfg,
		client:        &http.Client{Timeout: 30 * time.Second},
		accessToken:   cfg.WhatsAppToken,
		phoneNumberID: cfg.WhatsAppPhoneNumberID,
		apiURL:        cfg.WhatsAppURL,
	}
}

// EnviarVoucherGanador envía voucher cuando el cliente gana
func (w *WhatsAppService) EnviarVoucherGanador(cliente *models.Cliente, voucher *models.Voucher) error {
	if !w.isConfigured() {
		log.Printf(" WhatsApp no configurado, simulando envío de voucher ganador para %s", cliente.Telefono)
		return nil
	}

	templates := w.config.GetWhatsAppTemplates()
	templateName := templates["voucher_ganador"]

	message := models.WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               w.formatPhoneNumber(cliente.Telefono),
		Type:             "template",
		Template: &models.Template{
			Name:     templateName,
			Language: models.Language{Code: "es"},
			Components: []models.Component{
				{
					Type: "body",
					Parameters: []models.Parameter{
						{Type: "text", Text: cliente.Nombre},
						{Type: "text", Text: voucher.Codigo},
						{Type: "text", Text: fmt.Sprintf("%d%%", voucher.Descuento)},
						{Type: "text", Text: voucher.FechaVencimiento.Format("02/01/2006")},
					},
				},
			},
		},
	}

	return w.sendMessage(message)
}

// EnviarVoucherPerdedor envía voucher cuando el cliente pierde
func (w *WhatsAppService) EnviarVoucherPerdedor(cliente *models.Cliente, voucher *models.Voucher) error {
	if !w.isConfigured() {
		log.Printf("⚠️  WhatsApp no configurado, simulando envío de voucher perdedor para %s", cliente.Telefono)
		return nil
	}

	templates := w.config.GetWhatsAppTemplates()
	templateName := templates["voucher_perdedor"]

	message := models.WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               w.formatPhoneNumber(cliente.Telefono),
		Type:             "template",
		Template: &models.Template{
			Name:     templateName,
			Language: models.Language{Code: "es"},
			Components: []models.Component{
				{
					Type: "body",
					Parameters: []models.Parameter{
						{Type: "text", Text: cliente.Nombre},
						{Type: "text", Text: voucher.Codigo},
						{Type: "text", Text: fmt.Sprintf("%d%%", voucher.Descuento)},
						{Type: "text", Text: voucher.FechaVencimiento.Format("02/01/2006")},
					},
				},
			},
		},
	}

	return w.sendMessage(message)
}

// EnviarMensajeMarketing envía mensajes promocionales
func (w *WhatsAppService) EnviarMensajeMarketing(cliente *models.Cliente, mensaje string, codigoVoucher string) error {
	if !w.isConfigured() {
		log.Printf("⚠️  WhatsApp no configurado, simulando envío de marketing para %s", cliente.Telefono)
		return nil
	}

	// Para marketing, usar mensaje de texto simple (más flexible)
	mensajeCompleto := fmt.Sprintf("🧀 *CheeseHouse* 🧀\n\n%s\n\n🎁 *Código: %s*\n\n¡Te esperamos!",
		mensaje, codigoVoucher)

	message := models.WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               w.formatPhoneNumber(cliente.Telefono),
		Type:             "text",
		Text: &models.TextBody{
			Body: mensajeCompleto,
		},
	}

	return w.sendMessage(message)
}

// EnviarRespuestaAutomatica envía respuesta automática a pedidos
func (w *WhatsAppService) EnviarRespuestaAutomatica(telefono string, nombreCliente string) error {
	if !w.isConfigured() {
		log.Printf("⚠️  WhatsApp no configurado, simulando respuesta automática para %s", telefono)
		return nil
	}

	mensaje := fmt.Sprintf("¡Hola %s! 👋\n\n🧀 Gracias por contactar *CheeseHouse*\n\n⏰ Te responderemos en breve\n📞 O puedes llamarnos directamente\n\n¡Gracias por elegirnos! 🧀", nombreCliente)

	message := models.WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               w.formatPhoneNumber(telefono),
		Type:             "text",
		Text: &models.TextBody{
			Body: mensaje,
		},
	}

	return w.sendMessage(message)
}

// sendMessage envía un mensaje a WhatsApp API
func (w *WhatsAppService) sendMessage(message models.WhatsAppMessage) error {
	url := fmt.Sprintf("%s/%s/messages", w.apiURL, w.phoneNumberID)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error al serializar mensaje: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error al crear request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("📱 Enviando WhatsApp a %s: %s", message.To, string(jsonData))

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("error al enviar mensaje: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("WhatsApp API error %d: %v", resp.StatusCode, errorResp)
	}

	// Leer respuesta de éxito
	var successResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&successResp); err == nil {
		log.Printf("✅ WhatsApp enviado exitosamente: %v", successResp)
	}

	return nil
}

// ProcesarMensajeEntrante procesa mensajes recibidos por webhook
func (w *WhatsAppService) ProcesarMensajeEntrante(webhook models.WhatsAppWebhookMessage) []models.Pedido {
	var pedidos []models.Pedido

	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				for _, message := range change.Value.Messages {
					if message.Type == "text" {
						pedido := models.Pedido{
							Telefono:  w.normalizePhoneNumber(message.From),
							Mensaje:   message.Text.Body,
							Estado:    "pendiente",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}

						// Extraer nombre del contacto si está disponible
						for _, contact := range change.Value.Contacts {
							if contact.WaID == message.From {
								// Usar el nombre como nota por ahora
								pedido.Notas = fmt.Sprintf("Nombre WhatsApp: %s", contact.Profile.Name)
								break
							}
						}

						pedidos = append(pedidos, pedido)

						log.Printf("📨 Mensaje recibido de %s: %s", pedido.Telefono, pedido.Mensaje)
					}
				}
			}
		}
	}

	return pedidos
}

// formatPhoneNumber formatea número para WhatsApp API (sin +)
func (w *WhatsAppService) formatPhoneNumber(phone string) string {
	// WhatsApp API espera números sin el símbolo +
	return strings.TrimPrefix(phone, "+")
}

// normalizePhoneNumber normaliza número recibido para guardar en BD
func (w *WhatsAppService) normalizePhoneNumber(phone string) string {
	// Asegurar que tenga el prefijo +
	if !strings.HasPrefix(phone, "+") {
		return "+" + phone
	}
	return phone
}

// ValidarTelefonoArgentino valida formato de teléfono argentino
func (w *WhatsAppService) ValidarTelefonoArgentino(telefono string) error {
	validation := w.config.GetPhoneValidation()

	// Remover espacios y caracteres especiales
	cleanPhone := strings.ReplaceAll(telefono, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")

	// Verificar longitud
	if len(cleanPhone) < validation.MinLength || len(cleanPhone) > validation.MaxLength {
		return fmt.Errorf("número de teléfono debe tener entre %d y %d dígitos",
			validation.MinLength, validation.MaxLength)
	}

	// Verificar que empiece con +54 (Argentina) o permitir internacionales
	if !strings.HasPrefix(cleanPhone, validation.CountryCode) {
		if !validation.AllowIntl {
			return fmt.Errorf("número debe ser argentino (+54)")
		}
		// Si permite internacionales, verificar que empiece con +
		if !strings.HasPrefix(cleanPhone, "+") {
			return fmt.Errorf("número internacional debe empezar con +")
		}
	} else {
		// Es argentino, verificar código de área
		withoutCountryCode := strings.TrimPrefix(cleanPhone, validation.CountryCode)

		isValidAreaCode := false
		for _, areaCode := range validation.AreaCodes {
			if strings.HasPrefix(withoutCountryCode, areaCode) {
				isValidAreaCode = true
				break
			}
		}

		if !isValidAreaCode && len(withoutCountryCode) < 10 {
			return fmt.Errorf("código de área no válido para Argentina")
		}
	}

	return nil
}

// NormalizarTelefono normaliza y formatea un teléfono
func (w *WhatsAppService) NormalizarTelefono(telefono string) string {
	// Remover caracteres especiales
	cleanPhone := strings.ReplaceAll(telefono, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")

	// Asegurar que empiece con +
	if !strings.HasPrefix(cleanPhone, "+") {
		// Asumir argentino si no tiene prefijo internacional
		if len(cleanPhone) >= 10 {
			cleanPhone = "+54" + cleanPhone
		}
	}

	return cleanPhone
}

// isConfigured verifica si WhatsApp está configurado
func (w *WhatsAppService) isConfigured() bool {
	return w.accessToken != "" && w.phoneNumberID != ""
}

// GetStatus retorna el estado de configuración de WhatsApp
func (w *WhatsAppService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"configured":      w.isConfigured(),
		"access_token":    w.accessToken != "",
		"phone_number_id": w.phoneNumberID != "",
		"api_url":         w.apiURL,
	}
}

// TestConnection prueba la conexión con WhatsApp API
func (w *WhatsAppService) TestConnection() error {
	if !w.isConfigured() {
		return fmt.Errorf("WhatsApp no está configurado")
	}

	url := fmt.Sprintf("%s/%s", w.apiURL, w.phoneNumberID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error al crear request de test: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("error al conectar con WhatsApp API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WhatsApp API respondió con código: %d", resp.StatusCode)
	}

	log.Println("✅ Conexión con WhatsApp API exitosa")
	return nil
}

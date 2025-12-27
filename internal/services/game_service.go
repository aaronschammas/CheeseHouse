package services

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"CheeseHouse/internal/config"
	"CheeseHouse/internal/models"
	"CheeseHouse/internal/repository"
)

// GameService maneja la lógica del juego de timing de CheeseHouse
type GameService struct {
	config          *config.Config
	clienteRepo     *repository.ClienteRepository
	voucherRepo     repository.VoucherRepository
	whatsappService *WhatsAppService
}

// NewGameService crea una nueva instancia del servicio de juego
func NewGameService(
	config *config.Config,
	clienteRepo *repository.ClienteRepository,
	voucherRepo repository.VoucherRepository,
	whatsappService *WhatsAppService,
) *GameService {
	return &GameService{
		config:          config,
		clienteRepo:     clienteRepo,
		voucherRepo:     voucherRepo,
		whatsappService: whatsappService,
	}
}

// ProcesarResultadoJuego procesa el resultado completo del juego
func (g *GameService) ProcesarResultadoJuego(gameResult models.GameResult) (*models.VoucherResponse, error) {
	log.Printf("🎮 Procesando juego para %s %s - Tel: %s",
		gameResult.ClienteData.Nombre,
		gameResult.ClienteData.Apellido,
		gameResult.ClienteData.Telefono)

	// 1. Validar teléfono
	telefonoNormalizado := g.whatsappService.NormalizarTelefono(gameResult.ClienteData.Telefono)
	if err := g.whatsappService.ValidarTelefonoArgentino(telefonoNormalizado); err != nil {
		log.Printf("❌ Teléfono inválido: %v", err)
		return &models.VoucherResponse{
			Success: false,
			Message: "Número de teléfono no válido: " + err.Error(),
		}, nil
	}

	// 2. Validar datos del juego
	if err := g.validarDatosJuego(gameResult.Resultado); err != nil {
		return &models.VoucherResponse{
			Success: false,
			Message: "Datos del juego inválidos: " + err.Error(),
		}, nil
	}

	// 3. Determinar si ganó o perdió
	gano := g.determinarSiGano(gameResult.Resultado)
	log.Printf("🎯 Objetivo: %.1fs, Obtenido: %.1fs, Ganó: %t",
		gameResult.Resultado.TiempoObjetivo,
		gameResult.Resultado.TiempoObtenido,
		gano)

	// 4. Crear o buscar cliente
	cliente, esNuevo, err := g.crearOBuscarCliente(models.ClienteData{
		Nombre:   gameResult.ClienteData.Nombre,
		Apellido: gameResult.ClienteData.Apellido,
		Telefono: telefonoNormalizado,
	})
	if err != nil {
		return &models.VoucherResponse{
			Success: false,
			Message: "Error al procesar cliente: " + err.Error(),
		}, nil
	}

	// 5. Verificar si necesita aprobación (≥3 juegos)
	necesitaAprobacion := cliente.TotalJuegos >= g.config.Game.GamesRequireApproval

	if necesitaAprobacion {
		log.Printf("⚠️  Cliente %s necesita aprobación para juego #%d",
			cliente.Telefono, cliente.TotalJuegos+1)

		return &models.VoucherResponse{
			Success:            false,
			Message:            "Este cliente necesita aprobación de un empleado para seguir jugando",
			NecesitaAprobacion: true,
			ClienteID:          cliente.ID,
		}, nil
	}

	// 6. Crear voucher y actualizar estadísticas
	voucher, err := g.crearVoucherYActualizarCliente(cliente, gano)
	if err != nil {
		return &models.VoucherResponse{
			Success: false,
			Message: "Error al crear voucher: " + err.Error(),
		}, nil
	}

	// 7. Enviar WhatsApp
	go g.enviarWhatsAppAsync(cliente, voucher, gano)

	// 8. Retornar respuesta exitosa
	return &models.VoucherResponse{
		Success:            true,
		Message:            g.generarMensajeExito(gano, voucher.Descuento),
		Codigo:             voucher.Codigo,
		Descuento:          voucher.Descuento,
		FechaVencimiento:   voucher.FechaVencimiento.Format("02/01/2006"),
		ClienteID:          cliente.ID,
		EsClienteNuevo:     esNuevo,
		NecesitaAprobacion: false,
	}, nil
}

// GenerarTiempoObjetivo genera un tiempo objetivo aleatorio
func (g *GameService) GenerarTiempoObjetivo() float64 {
	rand.Seed(time.Now().UnixNano())
	min := g.config.Game.MinTargetTime
	max := g.config.Game.MaxTargetTime

	// Generar número aleatorio entre min y max con 1 decimal
	tiempo := min + rand.Float64()*(max-min)
	return math.Round(tiempo*10) / 10 // Redondear a 1 decimal
}

// determinarSiGano determina si el jugador ganó basado en la tolerancia
func (g *GameService) determinarSiGano(resultado models.Resultado) bool {
	diferencia := math.Abs(resultado.TiempoObtenido - resultado.TiempoObjetivo)
	return diferencia <= g.config.Game.Tolerance
}

// validarDatosJuego valida que los datos del juego sean coherentes
func (g *GameService) validarDatosJuego(resultado models.Resultado) error {
	if resultado.TiempoObjetivo < g.config.Game.MinTargetTime ||
		resultado.TiempoObjetivo > g.config.Game.MaxTargetTime {
		return fmt.Errorf("tiempo objetivo fuera de rango (%.1f-%.1fs)",
			g.config.Game.MinTargetTime, g.config.Game.MaxTargetTime)
	}

	if resultado.TiempoObtenido < 0 || resultado.TiempoObtenido > 30 {
		return fmt.Errorf("tiempo obtenido sospechoso: %.2fs", resultado.TiempoObtenido)
	}

	// Validación anti-trampa: diferencias muy pequeñas son sospechosas
	diferencia := math.Abs(resultado.TiempoObtenido - resultado.TiempoObjetivo)
	if diferencia < 0.01 && diferencia > 0 {
		log.Printf("⚠️  Diferencia sospechosamente pequeña: %.3fs", diferencia)
		// No bloquear, pero loguear para auditoría
	}

	return nil
}

// crearOBuscarCliente crea un cliente nuevo o busca uno existente
func (g *GameService) crearOBuscarCliente(clienteData models.ClienteData) (*models.Cliente, bool, error) {
	// Buscar cliente existente por teléfono
	cliente, err := g.clienteRepo.BuscarPorTelefono(clienteData.Telefono)
	if err != nil {
		// Si no existe, crear nuevo cliente
		nuevoCliente := &models.Cliente{
			Nombre:         clienteData.Nombre,
			Apellido:       clienteData.Apellido,
			Telefono:       clienteData.Telefono,
			FechaRegistro:  time.Now(),
			TotalJuegos:    0,
			JuegosGanados:  0,
			JuegosPerdidos: 0,
			Estado:         "activo",
		}

		if err := g.clienteRepo.Crear(nuevoCliente); err != nil {
			return nil, false, fmt.Errorf("error al crear cliente: %w", err)
		}

		log.Printf("✨ Cliente nuevo creado: %s %s (%s)",
			nuevoCliente.Nombre, nuevoCliente.Apellido, nuevoCliente.Telefono)

		return nuevoCliente, true, nil
	}

	// Cliente existente, actualizar datos si han cambiado
	actualizado := false
	if cliente.Nombre != clienteData.Nombre || cliente.Apellido != clienteData.Apellido {
		cliente.Nombre = clienteData.Nombre
		cliente.Apellido = clienteData.Apellido
		actualizado = true
	}

	if actualizado {
		if err := g.clienteRepo.Actualizar(cliente); err != nil {
			log.Printf("⚠️  Error al actualizar datos del cliente: %v", err)
		} else {
			log.Printf("📝 Datos del cliente actualizados: %s %s", cliente.Nombre, cliente.Apellido)
		}
	}

	return cliente, false, nil
}

// crearVoucherYActualizarCliente crea el voucher y actualiza las estadísticas del cliente
func (g *GameService) crearVoucherYActualizarCliente(cliente *models.Cliente, gano bool) (*models.Voucher, error) {
	// Determinar descuento
	var descuento int
	var tipo string
	if gano {
		descuento = g.config.Game.WinDiscount
		tipo = "juego_ganado"
	} else {
		descuento = g.config.Game.LoseDiscount
		tipo = "juego_perdido"
	}

	// Crear voucher
	voucher := &models.Voucher{
		Codigo:           g.generarCodigoVoucher(),
		ClienteID:        cliente.ID,
		Tipo:             tipo,
		Descuento:        descuento,
		Ganado:           &gano,
		FechaEmision:     time.Now(),
		FechaVencimiento: time.Now().AddDate(0, 0, g.config.Game.VoucherValidityDays),
		Usado:            false,
	}

	if err := g.voucherRepo.Crear(voucher); err != nil {
		return nil, fmt.Errorf("error al crear voucher: %w", err)
	}

	// Actualizar estadísticas del cliente
	cliente.TotalJuegos++
	hoy := time.Now()
	cliente.FechaUltimoJuego = &hoy

	if gano {
		cliente.JuegosGanados++
	} else {
		cliente.JuegosPerdidos++
	}

	if err := g.clienteRepo.Actualizar(cliente); err != nil {
		log.Printf("⚠️  Error al actualizar estadísticas del cliente: %v", err)
		// No es crítico, el voucher ya se creó
	}

	log.Printf("🎟️  Voucher creado: %s (%d%% descuento) para %s",
		voucher.Codigo, voucher.Descuento, cliente.Telefono)

	return voucher, nil
}

// generarCodigoVoucher genera un código único para el voucher
func (g *GameService) generarCodigoVoucher() string {
	prefix := g.config.GenerateVoucherCode() // "CH"
	timestamp := time.Now().Unix() % 100000  // Últimos 5 dígitos del timestamp
	random := rand.Intn(1000)                // Número aleatorio 0-999

	return fmt.Sprintf("%s%05d%03d", prefix, timestamp, random)
}

// enviarWhatsAppAsync envía WhatsApp de forma asíncrona
func (g *GameService) enviarWhatsAppAsync(cliente *models.Cliente, voucher *models.Voucher, gano bool) {
	var err error

	if gano {
		err = g.whatsappService.EnviarVoucherGanador(cliente, voucher)
	} else {
		err = g.whatsappService.EnviarVoucherPerdedor(cliente, voucher)
	}

	if err != nil {
		log.Printf("❌ Error enviando WhatsApp a %s: %v", cliente.Telefono, err)
		// TODO: Marcar voucher para reintento de envío
	} else {
		log.Printf("📱 WhatsApp enviado exitosamente a %s", cliente.Telefono)
	}
}

// generarMensajeExito genera mensaje de éxito para la respuesta
func (g *GameService) generarMensajeExito(gano bool, descuento int) string {
	if gano {
		return fmt.Sprintf("¡Felicitaciones! Ganaste un %d%% de descuento. Te enviamos el código por WhatsApp.", descuento)
	}
	return fmt.Sprintf("¡Casi! No te preocupes, tienes un %d%% de descuento de consolación. Revisa tu WhatsApp.", descuento)
}

// GetEstadisticasGenerales obtiene estadísticas generales del juego
func (g *GameService) GetEstadisticasGenerales() (*models.EstadisticasGenerales, error) {
	stats, err := g.clienteRepo.GetEstadisticasGenerales()
	if err != nil {
		return nil, fmt.Errorf("error al obtener estadísticas: %w", err)
	}

	// Obtener estadísticas de vouchers
	vouchersActivos, err := g.voucherRepo.ContarVouchersActivos()
	if err != nil {
		log.Printf("⚠️  Error al contar vouchers activos: %v", err)
	} else {
		stats.VouchersActivos = vouchersActivos
	}

	vouchersVencidos, err := g.voucherRepo.ContarVouchersVencidos()
	if err != nil {
		log.Printf("⚠️  Error al contar vouchers vencidos: %v", err)
	} else {
		stats.VouchersVencidos = vouchersVencidos
	}

	return stats, nil
}

// GetEstadisticasPorPeriodo obtiene estadísticas por período
func (g *GameService) GetEstadisticasPorPeriodo(dias int) ([]*models.EstadisticasPorPeriodo, error) {
	return g.voucherRepo.GetEstadisticasPorPeriodo(dias)
}

// ValidarAprobacionJuego valida si un cliente puede seguir jugando
func (g *GameService) ValidarAprobacionJuego(clienteID uint, empleadoID uint) error {
	cliente, err := g.clienteRepo.BuscarPorID(clienteID)
	if err != nil {
		return fmt.Errorf("cliente no encontrado: %w", err)
	}

	if cliente.TotalJuegos < g.config.Game.GamesRequireApproval {
		return fmt.Errorf("cliente no necesita aprobación")
	}

	// Actualizar fecha de última aprobación o similar lógica
	// Por ahora, simplemente loguear la aprobación
	log.Printf("✅ Empleado ID %d aprobó juego para cliente %s (%s)",
		empleadoID, cliente.Telefono, cliente.Nombre)

	return nil
}

// GetClientePorTelefono busca un cliente por teléfono (para consultas)
func (g *GameService) GetClientePorTelefono(telefono string) (*models.ClienteConEstadisticas, error) {
	telefonoNormalizado := g.whatsappService.NormalizarTelefono(telefono)

	cliente, err := g.clienteRepo.BuscarPorTelefono(telefonoNormalizado)
	if err != nil {
		return nil, fmt.Errorf("cliente no encontrado: %w", err)
	}

	// Obtener estadísticas completas
	return g.clienteRepo.GetClienteConEstadisticas(cliente.ID)
}

// GetConfiguracionJuego retorna la configuración actual del juego
func (g *GameService) GetConfiguracionJuego() map[string]interface{} {
	return map[string]interface{}{
		"tolerancia":         g.config.Game.Tolerance,
		"descuento_ganador":  g.config.Game.WinDiscount,
		"descuento_perdedor": g.config.Game.LoseDiscount,
		"tiempo_min":         g.config.Game.MinTargetTime,
		"tiempo_max":         g.config.Game.MaxTargetTime,
		"validez_voucher":    g.config.Game.VoucherValidityDays,
		"juegos_aprobacion":  g.config.Game.GamesRequireApproval,
		"restaurante":        g.config.RestaurantName,
	}
}

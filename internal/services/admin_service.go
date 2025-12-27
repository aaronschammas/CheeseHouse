package services

import (
	"fmt"
	"log"
	"time"

	"CheeseHouse/internal/models"
	"CheeseHouse/internal/repository"
)

// AdminService maneja las operaciones administrativas de CheeseHouse
type AdminService struct {
	clienteRepo     repository.ClienteRepository
	voucherRepo     repository.VoucherRepository
	whatsappService *WhatsAppService
}

// NewAdminService crea una nueva instancia del servicio administrativo
func NewAdminService(
	clienteRepo repository.ClienteRepository,
	voucherRepo repository.VoucherRepository,
	whatsappService *WhatsAppService,
) *AdminService {
	return &AdminService{
		clienteRepo:     clienteRepo,
		voucherRepo:     voucherRepo,
		whatsappService: whatsappService,
	}
}

// GetDashboardData obtiene todos los datos para el dashboard
func (a *AdminService) GetDashboardData() (map[string]interface{}, error) {
	// Estadísticas generales
	stats, err := a.clienteRepo.GetEstadisticasGenerales()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas: %w", err)
	}

	// Vouchers activos
	vouchersActivos, err := a.voucherRepo.ContarVouchersActivos()
	if err != nil {
		log.Printf("⚠️  Error contando vouchers activos: %v", err)
	} else {
		stats.VouchersActivos = vouchersActivos
	}

	// Vouchers por vencer (próximos 7 días)
	vouchersPorVencer, err := a.voucherRepo.GetVouchersPorVencer(7)
	if err != nil {
		log.Printf("⚠️  Error obteniendo vouchers por vencer: %v", err)
		vouchersPorVencer = []*models.Voucher{}
	}

	// Top 10 clientes más activos
	topClientes, err := a.clienteRepo.GetTopClientes(10)
	if err != nil {
		log.Printf("⚠️  Error obteniendo top clientes: %v", err)
		topClientes = []*models.ClienteConEstadisticas{}
	}

	// Estadísticas de los últimos 7 días
	estadisticasPeriodo, err := a.voucherRepo.GetEstadisticasPorPeriodo(7)
	if err != nil {
		log.Printf("⚠️  Error obteniendo estadísticas por período: %v", err)
		estadisticasPeriodo = []*models.EstadisticasPorPeriodo{}
	}

	return map[string]interface{}{
		"estadisticas_generales": stats,
		"vouchers_por_vencer":    vouchersPorVencer,
		"top_clientes":           topClientes,
		"estadisticas_periodo":   estadisticasPeriodo,
		"whatsapp_status":        a.whatsappService.GetStatus(),
	}, nil
}

// CanjearVoucher canjea un voucher en caja
func (a *AdminService) CanjearVoucher(codigo string, empleadoID uint) (*models.CanjearVoucherResponse, error) {
	log.Printf("🎟️  Canjeando voucher: %s por empleado ID: %d", codigo, empleadoID)

	// Buscar voucher
	voucher, err := a.voucherRepo.BuscarPorCodigo(codigo)
	if err != nil {
		return &models.CanjearVoucherResponse{
			Success: false,
			Message: "Código de voucher no válido",
		}, nil
	}

	// Verificar si ya fue usado
	if voucher.Usado {
		return &models.CanjearVoucherResponse{
			Success:   false,
			Message:   "Este voucher ya fue utilizado",
			Descuento: voucher.Descuento,
		}, nil
	}

	// Verificar vencimiento
	if voucher.FechaVencimiento.Before(time.Now()) {
		return &models.CanjearVoucherResponse{
			Success:   false,
			Message:   "Este voucher está vencido",
			Descuento: voucher.Descuento,
		}, nil
	}

	// Marcar como usado
	voucher.Usado = true
	now := time.Now()
	voucher.FechaUso = &now
	voucher.UsuarioCanje = &empleadoID

	if err := a.voucherRepo.Actualizar(voucher); err != nil {
		return &models.CanjearVoucherResponse{
			Success: false,
			Message: "Error interno procesando canje",
		}, nil
	}

	// Obtener datos del cliente
	cliente, err := a.clienteRepo.BuscarPorID(voucher.ClienteID)
	if err != nil {
		log.Printf("⚠️  Error obteniendo cliente para voucher %s: %v", codigo, err)
	}

	clienteNombre := "Cliente"
	if cliente != nil {
		clienteNombre = fmt.Sprintf("%s %s", cliente.Nombre, cliente.Apellido)
	}

	log.Printf("✅ Voucher %s canjeado exitosamente (%d%% descuento) para %s",
		codigo, voucher.Descuento, clienteNombre)

	return &models.CanjearVoucherResponse{
		Success:   true,
		Message:   "Voucher canjeado correctamente",
		Descuento: voucher.Descuento,
		Cliente:   clienteNombre,
	}, nil
}

// GetClientes obtiene lista de clientes con filtros
func (a *AdminService) GetClientes(filtros map[string]interface{}) ([]*models.ClienteConEstadisticas, error) {
	return a.clienteRepo.ListarConEstadisticas(filtros)
}

// GetClienteDetalle obtiene detalle completo de un cliente
func (a *AdminService) GetClienteDetalle(clienteID uint) (*models.ClienteConEstadisticas, error) {
	return a.clienteRepo.GetClienteConEstadisticas(clienteID)
}

// GetVouchers obtiene lista de vouchers con filtros
func (a *AdminService) GetVouchers(filtros map[string]interface{}) ([]*models.Voucher, error) {
	return a.voucherRepo.ListarConFiltros(filtros)
}

// CrearCampana crea una nueva campaña promocional
func (a *AdminService) CrearCampana(campana *models.CampanaClientesVouchers) error {
	// Validaciones
	if campana.Nombre == "" {
		return fmt.Errorf("nombre de campaña es requerido")
	}

	if campana.Descuento <= 0 || campana.Descuento > 100 {
		return fmt.Errorf("descuento debe estar entre 1 y 100")
	}

	if campana.FechaVencimiento.Before(time.Now()) {
		return fmt.Errorf("fecha de vencimiento debe ser futura")
	}

	// Crear campaña (implementar repository para campañas)
	log.Printf("📢 Creando campaña: %s (%d%% descuento)", campana.Nombre, campana.Descuento)

	// TODO: Implementar repository para campañas
	return fmt.Errorf("funcionalidad de campañas no implementada aún")
}

// EnviarCampana envía una campaña a clientes seleccionados
func (a *AdminService) EnviarCampana(campanaID uint, clientesIDs []uint) error {
	log.Printf("📢 Enviando campaña ID %d a %d clientes", campanaID, len(clientesIDs))

	// TODO: Implementar envío de campañas
	// 1. Obtener datos de la campaña
	// 2. Generar vouchers para cada cliente
	// 3. Enviar WhatsApp a cada cliente
	// 4. Registrar envíos en clientes_vouchers_envios

	return fmt.Errorf("funcionalidad de campañas no implementada aún")
}

// AprobarJuegoFrecuente aprueba que un cliente frecuente pueda seguir jugando
func (a *AdminService) AprobarJuegoFrecuente(clienteID uint, empleadoID uint) error {
	cliente, err := a.clienteRepo.BuscarPorID(clienteID)
	if err != nil {
		return fmt.Errorf("cliente no encontrado: %w", err)
	}

	if cliente.TotalJuegos < 3 {
		return fmt.Errorf("cliente no necesita aprobación (solo %d juegos)", cliente.TotalJuegos)
	}

	// Registrar aprobación
	log.Printf("✅ Empleado ID %d aprobó juegos para cliente %s %s (%s) - Total juegos: %d",
		empleadoID, cliente.Nombre, cliente.Apellido, cliente.Telefono, cliente.TotalJuegos)

	// TODO: Implementar sistema de aprobaciones en BD si es necesario
	// Por ahora solo logueamos la aprobación

	return nil
}

// GetClientesPendientesAprobacion obtiene clientes que necesitan aprobación
func (a *AdminService) GetClientesPendientesAprobacion() ([]*models.ClienteConEstadisticas, error) {
	filtros := map[string]interface{}{
		"min_juegos":  3,
		"jugaron_hoy": true,
	}

	return a.clienteRepo.ListarConEstadisticas(filtros)
}

// GetVouchersVencidos obtiene vouchers vencidos para análisis
func (a *AdminService) GetVouchersVencidos(dias int) ([]*models.Voucher, error) {
	return a.voucherRepo.GetVouchersVencidos(dias)
}

// GetReporteVentas genera reporte de "ventas" (vouchers canjeados)
func (a *AdminService) GetReporteVentas(fechaInicio, fechaFin time.Time) (map[string]interface{}, error) {
	vouchersCanjeados, err := a.voucherRepo.GetVouchersCanjeadosPorPeriodo(fechaInicio, fechaFin)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers canjeados: %w", err)
	}

	// Calcular métricas
	totalVouchers := len(vouchersCanjeados)
	totalDescuentos := 0
	descuentoPorDia := make(map[string]int)

	for _, voucher := range vouchersCanjeados {
		totalDescuentos += voucher.Descuento
		if voucher.FechaUso != nil {
			dia := voucher.FechaUso.Format("2006-01-02")
			descuentoPorDia[dia] += voucher.Descuento
		}
	}

	promedioDescuento := float64(0)
	if totalVouchers > 0 {
		promedioDescuento = float64(totalDescuentos) / float64(totalVouchers)
	}

	return map[string]interface{}{
		"periodo": map[string]string{
			"inicio": fechaInicio.Format("2006-01-02"),
			"fin":    fechaFin.Format("2006-01-02"),
		},
		"total_vouchers_canjeados": totalVouchers,
		"promedio_descuento":       promedioDescuento,
		"descuento_por_dia":        descuentoPorDia,
		"vouchers":                 vouchersCanjeados,
	}, nil
}

// GetEstadisticasDetalladas obtiene estadísticas detalladas para reportes
func (a *AdminService) GetEstadisticasDetalladas() (map[string]interface{}, error) {
	// Estadísticas generales
	statsGenerales, err := a.clienteRepo.GetEstadisticasGenerales()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas generales: %w", err)
	}

	// Estadísticas de vouchers
	vouchersStats := map[string]interface{}{
		"activos":    0,
		"vencidos":   0,
		"canjeados":  0,
		"pendientes": 0,
	}

	if activos, err := a.voucherRepo.ContarVouchersActivos(); err == nil {
		vouchersStats["activos"] = activos
	}

	if vencidos, err := a.voucherRepo.ContarVouchersVencidos(); err == nil {
		vouchersStats["vencidos"] = vencidos
	}

	if canjeados, err := a.voucherRepo.ContarVouchersCanjeados(); err == nil {
		vouchersStats["canjeados"] = canjeados
	}

	// Estadísticas por tipo de cliente
	clientesStats := map[string]interface{}{
		"nuevos":      0,
		"ocasionales": 0,
		"frecuentes":  0,
	}

	if nuevos, err := a.clienteRepo.ContarClientesPorTipo("nuevo"); err == nil {
		clientesStats["nuevos"] = nuevos
	}

	if ocasionales, err := a.clienteRepo.ContarClientesPorTipo("ocasional"); err == nil {
		clientesStats["ocasionales"] = ocasionales
	}

	if frecuentes, err := a.clienteRepo.ContarClientesPorTipo("frecuente"); err == nil {
		clientesStats["frecuentes"] = frecuentes
	}

	// Tendencia de los últimos 30 días
	tendencia, err := a.voucherRepo.GetEstadisticasPorPeriodo(30)
	if err != nil {
		log.Printf("⚠️  Error obteniendo tendencia: %v", err)
		tendencia = []*models.EstadisticasPorPeriodo{}
	}

	return map[string]interface{}{
		"resumen":           statsGenerales,
		"vouchers":          vouchersStats,
		"clientes":          clientesStats,
		"tendencia_30_dias": tendencia,
		"whatsapp":          a.whatsappService.GetStatus(),
		"generado_en":       time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// ProcesarPedidoWhatsApp procesa un pedido recibido por WhatsApp
func (a *AdminService) ProcesarPedidoWhatsApp(pedido *models.Pedido) error {
	log.Printf("📨 Procesando pedido de %s: %s", pedido.Telefono, pedido.Mensaje)

	// Buscar cliente por teléfono
	cliente, err := a.clienteRepo.BuscarPorTelefono(pedido.Telefono)
	if err != nil {
		log.Printf("⚠️  Cliente no encontrado para pedido: %s", pedido.Telefono)
		// Cliente nuevo, crear uno básico o manejar como pedido anónimo
	} else {
		pedido.ClienteID = cliente.ID
		log.Printf("👤 Pedido asociado al cliente: %s %s", cliente.Nombre, cliente.Apellido)
	}

	// TODO: Guardar pedido en base de datos cuando se implemente la tabla
	// Por ahora solo enviamos respuesta automática

	nombreCliente := "Cliente"
	if cliente != nil {
		nombreCliente = cliente.Nombre
	}

	// Enviar respuesta automática
	if err := a.whatsappService.EnviarRespuestaAutomatica(pedido.Telefono, nombreCliente); err != nil {
		log.Printf("❌ Error enviando respuesta automática: %v", err)
	}

	return nil
}

// GetConfiguracionSistema obtiene configuración del sistema para el admin
func (a *AdminService) GetConfiguracionSistema() map[string]interface{} {
	return map[string]interface{}{
		"restaurante": "CheeseHouse",
		"version":     "1.0.0",
		"ambiente":    "desarrollo", // TODO: obtener de config
		"whatsapp": map[string]interface{}{
			"configurado": a.whatsappService.GetStatus()["configured"],
			"estado":      "activo", // TODO: verificar estado real
		},
		"base_datos": map[string]interface{}{
			"tipo":   "MySQL",
			"estado": "conectada",
		},
		"juego": map[string]interface{}{
			"tolerancia":         0.3,
			"descuento_ganador":  30,
			"descuento_perdedor": 10,
			"validez_vouchers":   30,
		},
	}
}

// ExportarDatos exporta datos para backup (formato básico)
func (a *AdminService) ExportarDatos(tipoExport string) (map[string]interface{}, error) {
	resultado := make(map[string]interface{})

	switch tipoExport {
	case "clientes":
		clientes, err := a.clienteRepo.ListarTodos()
		if err != nil {
			return nil, fmt.Errorf("error exportando clientes: %w", err)
		}
		resultado["clientes"] = clientes
		resultado["total"] = len(clientes)

	case "vouchers":
		vouchers, err := a.voucherRepo.ListarTodos()
		if err != nil {
			return nil, fmt.Errorf("error exportando vouchers: %w", err)
		}
		resultado["vouchers"] = vouchers
		resultado["total"] = len(vouchers)

	case "completo":
		// Exportar todo
		clientes, _ := a.clienteRepo.ListarTodos()
		vouchers, _ := a.voucherRepo.ListarTodos()
		estadisticas, _ := a.GetEstadisticasDetalladas()

		resultado["clientes"] = clientes
		resultado["vouchers"] = vouchers
		resultado["estadisticas"] = estadisticas
		resultado["exportado_en"] = time.Now().Format("2006-01-02 15:04:05")

	default:
		return nil, fmt.Errorf("tipo de export no válido: %s", tipoExport)
	}

	return resultado, nil
}

// LimpiarVouchersVencidos marca vouchers vencidos como tal (mantenimiento)
func (a *AdminService) LimpiarVouchersVencidos() (int, error) {
	return a.voucherRepo.MarcarVouchersVencidos()
}

// GetAlertasOperativas obtiene alertas para el dashboard
func (a *AdminService) GetAlertasOperativas() []map[string]interface{} {
	var alertas []map[string]interface{}

	// Verificar vouchers por vencer (próximos 3 días)
	vouchersPorVencer, err := a.voucherRepo.GetVouchersPorVencer(3)
	if err == nil && len(vouchersPorVencer) > 0 {
		alertas = append(alertas, map[string]interface{}{
			"tipo":        "warning",
			"titulo":      "Vouchers por vencer",
			"descripcion": fmt.Sprintf("%d vouchers vencen en los próximos 3 días", len(vouchersPorVencer)),
			"accion":      "revisar_vouchers",
		})
	}

	// Verificar estado de WhatsApp
	whatsappStatus := a.whatsappService.GetStatus()
	if !whatsappStatus["configured"].(bool) {
		alertas = append(alertas, map[string]interface{}{
			"tipo":        "error",
			"titulo":      "WhatsApp no configurado",
			"descripcion": "Los vouchers no se están enviando por WhatsApp",
			"accion":      "configurar_whatsapp",
		})
	}

	// Verificar clientes que necesitan aprobación
	clientesPendientes, err := a.GetClientesPendientesAprobacion()
	if err == nil && len(clientesPendientes) > 0 {
		alertas = append(alertas, map[string]interface{}{
			"tipo":        "info",
			"titulo":      "Clientes pendientes de aprobación",
			"descripcion": fmt.Sprintf("%d clientes frecuentes esperan aprobación", len(clientesPendientes)),
			"accion":      "revisar_aprobaciones",
		})
	}

	return alertas
}

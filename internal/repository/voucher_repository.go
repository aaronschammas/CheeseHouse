package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"CheeseHouse/internal/models"
)

// VoucherRepository define la interfaz para operaciones con vouchers
type VoucherRepository interface {
	// CRUD básico
	Crear(voucher *models.Voucher) error
	BuscarPorID(id uint) (*models.Voucher, error)
	BuscarPorCodigo(codigo string) (*models.Voucher, error)
	Actualizar(voucher *models.Voucher) error
	Eliminar(id uint) error
	ListarTodos() ([]*models.Voucher, error)
	ListarConFiltros(filtros map[string]interface{}) ([]*models.Voucher, error)

	// Consultas específicas de vouchers
	GetVouchersPorCliente(clienteID uint) ([]*models.Voucher, error)
	GetVouchersActivos() ([]*models.Voucher, error)
	GetVouchersVencidos(dias int) ([]*models.Voucher, error)
	GetVouchersPorVencer(dias int) ([]*models.Voucher, error)
	GetVouchersCanjeadosPorPeriodo(inicio, fin time.Time) ([]*models.Voucher, error)

	// Contadores y estadísticas
	ContarVouchersActivos() (int, error)
	ContarVouchersVencidos() (int, error)
	ContarVouchersCanjeados() (int, error)
	GetEstadisticasPorPeriodo(dias int) ([]*models.EstadisticasPorPeriodo, error)

	// Operaciones de mantenimiento
	MarcarVouchersVencidos() (int, error)
	LimpiarVouchersAntiguos(dias int) (int, error)
}

// voucherRepository implementación de VoucherRepository
type voucherRepository struct {
	db *gorm.DB
}

// NewVoucherRepository crea una nueva instancia del repositorio de vouchers
func NewVoucherRepository(db *gorm.DB) VoucherRepository {
	return &voucherRepository{db: db}
}

// Crear crea un nuevo voucher en la base de datos
func (r *voucherRepository) Crear(voucher *models.Voucher) error {
	if err := r.db.Create(voucher).Error; err != nil {
		return fmt.Errorf("error creando voucher: %w", err)
	}
	return nil
}

// BuscarPorID busca un voucher por su ID
func (r *voucherRepository) BuscarPorID(id uint) (*models.Voucher, error) {
	var voucher models.Voucher
	if err := r.db.Preload("Cliente").Preload("UsuarioQueCanje").First(&voucher, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("voucher con ID %d no encontrado", id)
		}
		return nil, fmt.Errorf("error buscando voucher: %w", err)
	}
	return &voucher, nil
}

// BuscarPorCodigo busca un voucher por su código único
func (r *voucherRepository) BuscarPorCodigo(codigo string) (*models.Voucher, error) {
	var voucher models.Voucher
	if err := r.db.Preload("Cliente").Preload("UsuarioQueCanje").
		Where("codigo = ?", codigo).First(&voucher).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("voucher con código %s no encontrado", codigo)
		}
		return nil, fmt.Errorf("error buscando voucher por código: %w", err)
	}
	return &voucher, nil
}

// Actualizar actualiza los datos de un voucher
func (r *voucherRepository) Actualizar(voucher *models.Voucher) error {
	if err := r.db.Save(voucher).Error; err != nil {
		return fmt.Errorf("error actualizando voucher: %w", err)
	}
	return nil
}

// Eliminar elimina un voucher (soft delete)
func (r *voucherRepository) Eliminar(id uint) error {
	if err := r.db.Delete(&models.Voucher{}, id).Error; err != nil {
		return fmt.Errorf("error eliminando voucher: %w", err)
	}
	return nil
}

// ListarTodos obtiene todos los vouchers
func (r *voucherRepository) ListarTodos() ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Preload("Cliente").Preload("UsuarioQueCanje").Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error listando vouchers: %w", err)
	}
	return vouchers, nil
}

// ListarConFiltros obtiene vouchers aplicando filtros
func (r *voucherRepository) ListarConFiltros(filtros map[string]interface{}) ([]*models.Voucher, error) {
	query := r.db.Preload("Cliente").Preload("UsuarioQueCanje")

	// Aplicar filtros
	if tipo, ok := filtros["tipo"]; ok {
		query = query.Where("tipo = ?", tipo)
	}

	if usado, ok := filtros["usado"]; ok {
		query = query.Where("usado = ?", usado)
	}

	if clienteID, ok := filtros["cliente_id"]; ok {
		query = query.Where("cliente_id = ?", clienteID)
	}

	if ganado, ok := filtros["ganado"]; ok {
		query = query.Where("ganado = ?", ganado)
	}

	if fechaDesde, ok := filtros["fecha_desde"]; ok {
		query = query.Where("fecha_emision >= ?", fechaDesde)
	}

	if fechaHasta, ok := filtros["fecha_hasta"]; ok {
		query = query.Where("fecha_emision <= ?", fechaHasta)
	}

	if vencido, ok := filtros["vencido"]; ok && vencido.(bool) {
		query = query.Where("fecha_vencimiento < CURDATE()")
	}

	if porVencer, ok := filtros["por_vencer_dias"]; ok {
		dias := porVencer.(int)
		query = query.Where("fecha_vencimiento BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL ? DAY)", dias)
	}

	// Ordenamiento
	orderBy := "created_at DESC"
	if order, ok := filtros["order_by"]; ok {
		orderBy = order.(string)
	}
	query = query.Order(orderBy)

	// Límite
	if limit, ok := filtros["limit"]; ok {
		query = query.Limit(limit.(int))
	}

	var vouchers []*models.Voucher
	if err := query.Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error listando vouchers con filtros: %w", err)
	}

	return vouchers, nil
}

// GetVouchersPorCliente obtiene todos los vouchers de un cliente específico
func (r *voucherRepository) GetVouchersPorCliente(clienteID uint) ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Where("cliente_id = ?", clienteID).
		Order("created_at DESC").
		Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers del cliente: %w", err)
	}
	return vouchers, nil
}

// GetVouchersActivos obtiene vouchers válidos y no usados
func (r *voucherRepository) GetVouchersActivos() ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Preload("Cliente").
		Where("usado = FALSE AND fecha_vencimiento >= CURDATE()").
		Order("fecha_vencimiento ASC").
		Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers activos: %w", err)
	}
	return vouchers, nil
}

// GetVouchersVencidos obtiene vouchers vencidos de los últimos X días
func (r *voucherRepository) GetVouchersVencidos(dias int) ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Preload("Cliente").
		Where("fecha_vencimiento < CURDATE() AND fecha_vencimiento >= DATE_SUB(CURDATE(), INTERVAL ? DAY)", dias).
		Order("fecha_vencimiento DESC").
		Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers vencidos: %w", err)
	}
	return vouchers, nil
}

// GetVouchersPorVencer obtiene vouchers que vencen en los próximos X días
func (r *voucherRepository) GetVouchersPorVencer(dias int) ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Preload("Cliente").
		Where("usado = FALSE AND fecha_vencimiento BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL ? DAY)", dias).
		Order("fecha_vencimiento ASC").
		Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers por vencer: %w", err)
	}
	return vouchers, nil
}

// GetVouchersCanjeadosPorPeriodo obtiene vouchers canjeados en un período
func (r *voucherRepository) GetVouchersCanjeadosPorPeriodo(inicio, fin time.Time) ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	if err := r.db.Preload("Cliente").Preload("UsuarioQueCanje").
		Where("usado = TRUE AND fecha_uso BETWEEN ? AND ?", inicio, fin).
		Order("fecha_uso DESC").
		Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers canjeados por período: %w", err)
	}
	return vouchers, nil
}

// ContarVouchersActivos cuenta vouchers válidos y no usados
func (r *voucherRepository) ContarVouchersActivos() (int, error) {
	var count int64
	if err := r.db.Model(&models.Voucher{}).
		Where("usado = FALSE AND fecha_vencimiento >= CURDATE()").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando vouchers activos: %w", err)
	}
	return int(count), nil
}

// ContarVouchersVencidos cuenta vouchers vencidos
func (r *voucherRepository) ContarVouchersVencidos() (int, error) {
	var count int64
	if err := r.db.Model(&models.Voucher{}).
		Where("fecha_vencimiento < CURDATE()").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando vouchers vencidos: %w", err)
	}
	return int(count), nil
}

// ContarVouchersCanjeados cuenta vouchers que han sido canjeados
func (r *voucherRepository) ContarVouchersCanjeados() (int, error) {
	var count int64
	if err := r.db.Model(&models.Voucher{}).
		Where("usado = TRUE").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando vouchers canjeados: %w", err)
	}
	return int(count), nil
}

// GetEstadisticasPorPeriodo obtiene estadísticas de juegos agrupadas por día
func (r *voucherRepository) GetEstadisticasPorPeriodo(dias int) ([]*models.EstadisticasPorPeriodo, error) {
	query := `
		SELECT 
			DATE(fecha_emision) as fecha,
			COUNT(CASE WHEN ganado = TRUE THEN 1 END) as victorias_dia,
			COUNT(CASE WHEN ganado = FALSE THEN 1 END) as derrotas_dia,
			COUNT(*) as total_juegos_dia,
			CASE 
				WHEN COUNT(*) > 0 THEN
					ROUND((COUNT(CASE WHEN ganado = TRUE THEN 1 END) / COUNT(*)) * 100, 2)
				ELSE 0
			END as porcentaje_victorias_dia
		FROM vouchers
		WHERE tipo IN ('juego_ganado', 'juego_perdido')
			AND fecha_emision >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY DATE(fecha_emision)
		ORDER BY fecha DESC
	`

	var estadisticas []*models.EstadisticasPorPeriodo
	if err := r.db.Raw(query, dias).Scan(&estadisticas).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas por período: %w", err)
	}

	return estadisticas, nil
}

// MarcarVouchersVencidos marca vouchers vencidos (operación de mantenimiento)
func (r *voucherRepository) MarcarVouchersVencidos() (int, error) {
	// Esta operación es más para logging/auditoría ya que MySQL maneja las fechas automáticamente
	var count int64
	if err := r.db.Model(&models.Voucher{}).
		Where("fecha_vencimiento < CURDATE() AND usado = FALSE").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando vouchers a marcar como vencidos: %w", err)
	}

	// Opcional: agregar campo "vencido" si queremos marcarlo explícitamente
	// UPDATE vouchers SET vencido = TRUE WHERE fecha_vencimiento < CURDATE() AND usado = FALSE

	return int(count), nil
}

// LimpiarVouchersAntiguos elimina vouchers muy antiguos (mantenimiento)
func (r *voucherRepository) LimpiarVouchersAntiguos(dias int) (int, error) {
	// Eliminar vouchers vencidos hace más de X días (para limpiar BD)
	result := r.db.Where("fecha_vencimiento < DATE_SUB(CURDATE(), INTERVAL ? DAY)", dias).
		Delete(&models.Voucher{})

	if result.Error != nil {
		return 0, fmt.Errorf("error limpiando vouchers antiguos: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// GetVouchersPorTipo obtiene vouchers filtrados por tipo
func (r *voucherRepository) GetVouchersPorTipo(tipo string, limit int) ([]*models.Voucher, error) {
	var vouchers []*models.Voucher
	query := r.db.Preload("Cliente").Where("tipo = ?", tipo).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&vouchers).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo vouchers por tipo: %w", err)
	}
	return vouchers, nil
}

// GetEstadisticasVouchersPorCliente obtiene estadísticas de vouchers agrupadas por cliente
func (r *voucherRepository) GetEstadisticasVouchersPorCliente() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			c.id,
			c.nombre,
			c.apellido,
			c.telefono,
			COUNT(v.id) as total_vouchers,
			COUNT(CASE WHEN v.usado = TRUE THEN 1 END) as vouchers_usados,
			COUNT(CASE WHEN v.usado = FALSE AND v.fecha_vencimiento >= CURDATE() THEN 1 END) as vouchers_activos,
			COUNT(CASE WHEN v.fecha_vencimiento < CURDATE() AND v.usado = FALSE THEN 1 END) as vouchers_vencidos,
			ROUND(AVG(v.descuento), 2) as promedio_descuento,
			MAX(v.created_at) as ultimo_voucher
		FROM clientes c
		LEFT JOIN vouchers v ON c.id = v.cliente_id
		WHERE c.estado = 'activo'
		GROUP BY c.id, c.nombre, c.apellido, c.telefono
		HAVING COUNT(v.id) > 0
		ORDER BY COUNT(v.id) DESC
	`

	var resultados []map[string]interface{}
	if err := r.db.Raw(query).Scan(&resultados).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas de vouchers por cliente: %w", err)
	}

	return resultados, nil
}

// ValidarCodigoUnico verifica si un código de voucher es único
func (r *voucherRepository) ValidarCodigoUnico(codigo string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Voucher{}).Where("codigo = ?", codigo).Count(&count).Error; err != nil {
		return false, fmt.Errorf("error validando código único: %w", err)
	}
	return count == 0, nil
}

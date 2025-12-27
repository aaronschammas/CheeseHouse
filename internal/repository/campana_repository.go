package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"CheeseHouse/internal/models"
)

// CampanaRepository define la interfaz para operaciones con campañas
type CampanaRepository interface {
	// CRUD básico
	Crear(campana *models.CampanaClientesVouchers) error
	BuscarPorID(id uint) (*models.CampanaClientesVouchers, error)
	Actualizar(campana *models.CampanaClientesVouchers) error
	Eliminar(id uint) error
	ListarTodas() ([]*models.CampanaClientesVouchers, error)
	ListarActivas() ([]*models.CampanaClientesVouchers, error)

	// Gestión de envíos
	CrearEnvio(envio *models.ClientesVouchersEnvios) error
	GetEnviosPorCampana(campanaID uint) ([]*models.ClientesVouchersEnvios, error)
	ActualizarEstadoEnvio(envioID uint, estado string, errorMsg string) error

	// Estadísticas de campañas
	GetEstadisticasCampana(campanaID uint) (map[string]interface{}, error)
	GetCampanasConEstadisticas() ([]map[string]interface{}, error)
}

// campanaRepository implementación de CampanaRepository
type campanaRepository struct {
	db *gorm.DB
}

// NewCampanaRepository crea una nueva instancia del repositorio de campañas
func NewCampanaRepository(db *gorm.DB) CampanaRepository {
	return &campanaRepository{db: db}
}

// Crear crea una nueva campaña
func (r *campanaRepository) Crear(campana *models.CampanaClientesVouchers) error {
	if err := r.db.Create(campana).Error; err != nil {
		return fmt.Errorf("error creando campaña: %w", err)
	}
	return nil
}

// BuscarPorID busca una campaña por su ID
func (r *campanaRepository) BuscarPorID(id uint) (*models.CampanaClientesVouchers, error) {
	var campana models.CampanaClientesVouchers
	if err := r.db.Preload("CreadoPor").Preload("Envios").First(&campana, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("campaña con ID %d no encontrada", id)
		}
		return nil, fmt.Errorf("error buscando campaña: %w", err)
	}
	return &campana, nil
}

// Actualizar actualiza una campaña
func (r *campanaRepository) Actualizar(campana *models.CampanaClientesVouchers) error {
	if err := r.db.Save(campana).Error; err != nil {
		return fmt.Errorf("error actualizando campaña: %w", err)
	}
	return nil
}

// Eliminar elimina una campaña (soft delete)
func (r *campanaRepository) Eliminar(id uint) error {
	if err := r.db.Delete(&models.CampanaClientesVouchers{}, id).Error; err != nil {
		return fmt.Errorf("error eliminando campaña: %w", err)
	}
	return nil
}

// ListarTodas obtiene todas las campañas
func (r *campanaRepository) ListarTodas() ([]*models.CampanaClientesVouchers, error) {
	var campanas []*models.CampanaClientesVouchers
	if err := r.db.Preload("CreadoPor").Order("created_at DESC").Find(&campanas).Error; err != nil {
		return nil, fmt.Errorf("error listando campañas: %w", err)
	}
	return campanas, nil
}

// ListarActivas obtiene campañas activas y no vencidas
func (r *campanaRepository) ListarActivas() ([]*models.CampanaClientesVouchers, error) {
	var campanas []*models.CampanaClientesVouchers
	if err := r.db.Preload("CreadoPor").
		Where("activa = TRUE AND fecha_vencimiento >= CURDATE()").
		Order("created_at DESC").
		Find(&campanas).Error; err != nil {
		return nil, fmt.Errorf("error listando campañas activas: %w", err)
	}
	return campanas, nil
}

// CrearEnvio registra un envío de campaña
func (r *campanaRepository) CrearEnvio(envio *models.ClientesVouchersEnvios) error {
	if err := r.db.Create(envio).Error; err != nil {
		return fmt.Errorf("error creando envío: %w", err)
	}
	return nil
}

// GetEnviosPorCampana obtiene todos los envíos de una campaña
func (r *campanaRepository) GetEnviosPorCampana(campanaID uint) ([]*models.ClientesVouchersEnvios, error) {
	var envios []*models.ClientesVouchersEnvios
	if err := r.db.Preload("Cliente").Preload("Voucher").
		Where("campaña_id = ?", campanaID).
		Order("enviado_at DESC").
		Find(&envios).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo envíos de campaña: %w", err)
	}
	return envios, nil
}

// ActualizarEstadoEnvio actualiza el estado de un envío
func (r *campanaRepository) ActualizarEstadoEnvio(envioID uint, estado string, errorMsg string) error {
	updates := map[string]interface{}{
		"estado": estado,
	}

	if errorMsg != "" {
		updates["error_mensaje"] = errorMsg
		// Incrementar contador de intentos si hay error
		if estado == "fallido" {
			if err := r.db.Model(&models.ClientesVouchersEnvios{}).
				Where("id = ?", envioID).
				Update("intentos_envio", gorm.Expr("intentos_envio + 1")).Error; err != nil {
				return fmt.Errorf("error incrementando intentos de envío: %w", err)
			}
		}
	}

	if err := r.db.Model(&models.ClientesVouchersEnvios{}).
		Where("id = ?", envioID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("error actualizando estado de envío: %w", err)
	}
	return nil
}

// GetEstadisticasCampana obtiene estadísticas detalladas de una campaña
func (r *campanaRepository) GetEstadisticasCampana(campanaID uint) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_envios,
			COUNT(CASE WHEN estado = 'enviado' THEN 1 END) as enviados,
			COUNT(CASE WHEN estado = 'entregado' THEN 1 END) as entregados,
			COUNT(CASE WHEN estado = 'fallido' THEN 1 END) as fallidos,
			COUNT(CASE WHEN voucher_id IS NOT NULL THEN 1 END) as vouchers_generados,
			AVG(intentos_envio) as promedio_intentos
		FROM clientes_vouchers_envios
		WHERE campaña_id = ?
	`

	var stats struct {
		TotalEnvios       int     `json:"total_envios"`
		Enviados          int     `json:"enviados"`
		Entregados        int     `json:"entregados"`
		Fallidos          int     `json:"fallidos"`
		VouchersGenerados int     `json:"vouchers_generados"`
		PromedioIntentos  float64 `json:"promedio_intentos"`
	}

	if err := r.db.Raw(query, campanaID).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas de campaña: %w", err)
	}

	// Calcular porcentajes
	resultado := map[string]interface{}{
		"total_envios":       stats.TotalEnvios,
		"enviados":           stats.Enviados,
		"entregados":         stats.Entregados,
		"fallidos":           stats.Fallidos,
		"vouchers_generados": stats.VouchersGenerados,
		"promedio_intentos":  stats.PromedioIntentos,
	}

	if stats.TotalEnvios > 0 {
		resultado["porcentaje_entrega"] = float64(stats.Entregados) / float64(stats.TotalEnvios) * 100
		resultado["porcentaje_fallo"] = float64(stats.Fallidos) / float64(stats.TotalEnvios) * 100
	}

	return resultado, nil
}

// GetCampanasConEstadisticas obtiene todas las campañas con sus estadísticas básicas
func (r *campanaRepository) GetCampanasConEstadisticas() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			c.id,
			c.nombre,
			c.descripcion,
			c.descuento,
			c.fecha_vencimiento,
			c.activa,
			c.created_at,
			u.nombre as creado_por,
			COUNT(e.id) as total_envios,
			COUNT(CASE WHEN e.estado = 'entregado' THEN 1 END) as entregados,
			COUNT(CASE WHEN e.estado = 'fallido' THEN 1 END) as fallidos
		FROM campañas_clientes_vouchers c
		LEFT JOIN usuarios u ON c.created_by = u.id
		LEFT JOIN clientes_vouchers_envios e ON c.id = e.campaña_id
		GROUP BY c.id, c.nombre, c.descripcion, c.descuento, c.fecha_vencimiento, 
				 c.activa, c.created_at, u.nombre
		ORDER BY c.created_at DESC
	`

	var campanas []map[string]interface{}
	if err := r.db.Raw(query).Scan(&campanas).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo campañas con estadísticas: %w", err)
	}

	return campanas, nil
}

// GetEnviosPendientesReintento obtiene envíos que fallaron y pueden ser reintentados
func (r *campanaRepository) GetEnviosPendientesReintento(maxIntentos int) ([]*models.ClientesVouchersEnvios, error) {
	var envios []*models.ClientesVouchersEnvios
	if err := r.db.Preload("Campana").Preload("Cliente").
		Where("estado = 'fallido' AND intentos_envio < ?", maxIntentos).
		Find(&envios).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo envíos pendientes de reintento: %w", err)
	}
	return envios, nil
}

// LimpiarCampanasAntiguas elimina campañas muy antiguas (mantenimiento)
func (r *campanaRepository) LimpiarCampanasAntiguas(diasAntiguedad int) (int, error) {
	// Eliminar campañas vencidas hace más de X días
	result := r.db.Where("fecha_vencimiento < DATE_SUB(CURDATE(), INTERVAL ? DAY)", diasAntiguedad).
		Delete(&models.CampanaClientesVouchers{})

	if result.Error != nil {
		return 0, fmt.Errorf("error limpiando campañas antigas: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// GetCampanasPorPeriodo obtiene campañas creadas en un período específico
func (r *campanaRepository) GetCampanasPorPeriodo(inicio, fin time.Time) ([]*models.CampanaClientesVouchers, error) {
	var campanas []*models.CampanaClientesVouchers
	if err := r.db.Preload("CreadoPor").
		Where("created_at BETWEEN ? AND ?", inicio, fin).
		Order("created_at DESC").
		Find(&campanas).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo campañas por período: %w", err)
	}
	return campanas, nil
}

package repository

import (
	"CheeseHouse/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type ClienteRepository struct {
	db *gorm.DB
}

func NewClienteRepository(db *gorm.DB) *ClienteRepository {
	return &ClienteRepository{db: db}
}

func (r *ClienteRepository) Create(cliente *models.Cliente) error {
	return r.db.Create(cliente).Error
}

func (r *ClienteRepository) GetByTelefono(telefono string) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.Preload("Juegos").Preload("Vouchers").Where("telefono = ?", telefono).First(&cliente).Error
	if err != nil {
		return nil, err
	}
	return &cliente, nil
}

func (r *ClienteRepository) GetByID(id uint) (*models.Cliente, error) {
	var cliente models.Cliente
	err := r.db.Preload("Juegos").Preload("Vouchers").First(&cliente, id).Error
	return &cliente, err
}

func (r *ClienteRepository) GetAll() ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.Preload("Juegos").Preload("Vouchers").Find(&clientes).Error
	return clientes, err
}

func (r *ClienteRepository) Update(cliente *models.Cliente) error {
	return r.db.Save(cliente).Error
}

func (r *ClienteRepository) Delete(id uint) error {
	return r.db.Delete(&models.Cliente{}, id).Error
}

func (r *ClienteRepository) ExistsByTelefono(telefono string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Cliente{}).Where("telefono = ?", telefono).Count(&count).Error
	return count > 0, err
}

func (r *ClienteRepository) GetClientesWithMultipleGames(minGames int) ([]models.Cliente, error) {
	var clientes []models.Cliente
	err := r.db.Preload("Juegos").Preload("Vouchers").
		Joins("LEFT JOIN juegos ON clientes.id = juegos.cliente_id").
		Group("clientes.id").
		Having("COUNT(juegos.id) >= ?", minGames).
		Find(&clientes).Error
	return clientes, err
}

func (r *ClienteRepository) GetEstadisticasGenerales() (*models.EstadisticasGenerales, error) {
	var stats models.EstadisticasGenerales

	// Total de clientes
	var totalClientes int64
	r.db.Model(&models.Cliente{}).Count(&totalClientes)
	stats.TotalClientes = int(totalClientes)

	// Sumar estadísticas de todos los clientes
	var totalPartidas, totalVictorias, totalDerrotas int64
	r.db.Model(&models.Cliente{}).Select("SUM(total_juegos)").Scan(&totalPartidas)
	r.db.Model(&models.Cliente{}).Select("SUM(juegos_ganados)").Scan(&totalVictorias)
	r.db.Model(&models.Cliente{}).Select("SUM(juegos_perdidos)").Scan(&totalDerrotas)

	stats.TotalPartidas = int(totalPartidas)
	stats.TotalVictorias = int(totalVictorias)
	stats.TotalDerrotas = int(totalDerrotas)

	if totalPartidas > 0 {
		stats.PorcentajeVictorias = float64(totalVictorias) / float64(totalPartidas) * 100
	}

	// Clientes que jugaron hoy (simplificado)
	var jugaronHoy int64
	r.db.Model(&models.Cliente{}).Where("fecha_ultimo_juego >= CURDATE()").Count(&jugaronHoy)
	stats.JugaronHoy = int(jugaronHoy)

	// Clientes frecuentes (más de 3 juegos)
	var clientesFrecuentes int64
	r.db.Model(&models.Cliente{}).Where("total_juegos > 3").Count(&clientesFrecuentes)
	stats.ClientesFrecuentes = int(clientesFrecuentes)

	return &stats, nil
}

func (r *ClienteRepository) GetClienteConEstadisticas(clienteID uint) (*models.ClienteConEstadisticas, error) {
	var cliente models.Cliente
	err := r.db.Preload("Vouchers").First(&cliente, clienteID).Error
	if err != nil {
		return nil, err
	}

	// Calcular estadísticas adicionales
	totalJuegos := cliente.TotalJuegos
	victorias := cliente.JuegosGanados

	// Calcular porcentaje de victorias personal
	var porcentajeVictorias float64
	if totalJuegos > 0 {
		porcentajeVictorias = float64(victorias) / float64(totalJuegos) * 100
	}

	// Determinar tipo de cliente
	tipoCliente := "nuevo"
	if totalJuegos > 10 {
		tipoCliente = "frecuente"
	} else if totalJuegos > 3 {
		tipoCliente = "ocasional"
	}

	// Contar vouchers por estado
	vouchersGenerados := len(cliente.Vouchers)
	vouchersUsados := 0
	vouchersPendientes := 0
	var ultimoVoucher *models.Voucher

	for _, voucher := range cliente.Vouchers {
		if voucher.Usado {
			vouchersUsados++
		} else {
			vouchersPendientes++
		}
		if ultimoVoucher == nil || voucher.FechaEmision.After(ultimoVoucher.FechaEmision) {
			ultimoVoucher = &voucher
		}
	}

	return &models.ClienteConEstadisticas{
		Cliente:                     cliente,
		VouchersGenerados:           vouchersGenerados,
		VouchersUsados:              vouchersUsados,
		VouchersPendientes:          vouchersPendientes,
		PorcentajeVictoriasPersonal: porcentajeVictorias,
		TipoCliente:                 tipoCliente,
		UltimoVoucher:               ultimoVoucher,
	}, nil
}

// Alias methods for compatibility with game_service.go
func (r *ClienteRepository) BuscarPorTelefono(telefono string) (*models.Cliente, error) {
	return r.GetByTelefono(telefono)
}

func (r *ClienteRepository) BuscarPorID(id uint) (*models.Cliente, error) {
	return r.GetByID(id)
}

func (r *ClienteRepository) Crear(cliente *models.Cliente) error {
	return r.Create(cliente)
}

func (r *ClienteRepository) Actualizar(cliente *models.Cliente) error {
	return r.Update(cliente)
}

// GetTopClientes obtiene los N clientes más activos
func (r *ClienteRepository) GetTopClientes(limit int) ([]*models.ClienteConEstadisticas, error) {
	var clientes []models.Cliente
	err := r.db.Preload("Vouchers").Order("total_juegos DESC").Limit(limit).Find(&clientes).Error
	if err != nil {
		return nil, err
	}

	var result []*models.ClienteConEstadisticas
	for _, cliente := range clientes {
		// Calcular estadísticas adicionales
		totalJuegos := cliente.TotalJuegos
		victorias := cliente.JuegosGanados

		// Calcular porcentaje de victorias personal
		var porcentajeVictorias float64
		if totalJuegos > 0 {
			porcentajeVictorias = float64(victorias) / float64(totalJuegos) * 100
		}

		// Determinar tipo de cliente
		tipoCliente := "nuevo"
		if totalJuegos > 10 {
			tipoCliente = "frecuente"
		} else if totalJuegos > 3 {
			tipoCliente = "ocasional"
		}

		// Contar vouchers por estado
		vouchersGenerados := len(cliente.Vouchers)
		vouchersUsados := 0
		vouchersPendientes := 0
		var ultimoVoucher *models.Voucher

		for _, voucher := range cliente.Vouchers {
			if voucher.Usado {
				vouchersUsados++
			} else {
				vouchersPendientes++
			}
			if ultimoVoucher == nil || voucher.FechaEmision.After(ultimoVoucher.FechaEmision) {
				ultimoVoucher = &voucher
			}
		}

		result = append(result, &models.ClienteConEstadisticas{
			Cliente:                     cliente,
			VouchersGenerados:           vouchersGenerados,
			VouchersUsados:              vouchersUsados,
			VouchersPendientes:          vouchersPendientes,
			PorcentajeVictoriasPersonal: porcentajeVictorias,
			TipoCliente:                 tipoCliente,
			UltimoVoucher:               ultimoVoucher,
		})
	}

	return result, nil
}

// ListarConEstadisticas lista clientes con estadísticas aplicando filtros
func (r *ClienteRepository) ListarConEstadisticas(filtros map[string]interface{}) ([]*models.ClienteConEstadisticas, error) {
	query := r.db.Preload("Vouchers")

	// Aplicar filtros
	if telefono, ok := filtros["telefono"].(string); ok && telefono != "" {
		query = query.Where("telefono LIKE ?", "%"+telefono+"%")
	}
	if nombre, ok := filtros["nombre"].(string); ok && nombre != "" {
		query = query.Where("nombre LIKE ? OR apellido LIKE ?", "%"+nombre+"%", "%"+nombre+"%")
	}
	if estado, ok := filtros["estado"].(string); ok && estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if tipoCliente, ok := filtros["tipo_cliente"].(string); ok && tipoCliente != "" {
		switch tipoCliente {
		case "nuevo":
			query = query.Where("total_juegos <= 3")
		case "ocasional":
			query = query.Where("total_juegos > 3 AND total_juegos <= 10")
		case "frecuente":
			query = query.Where("total_juegos > 10")
		}
	}

	var clientes []models.Cliente
	err := query.Find(&clientes).Error
	if err != nil {
		return nil, err
	}

	var result []*models.ClienteConEstadisticas
	for _, cliente := range clientes {
		// Calcular estadísticas adicionales
		totalJuegos := cliente.TotalJuegos
		victorias := cliente.JuegosGanados

		// Calcular porcentaje de victorias personal
		var porcentajeVictorias float64
		if totalJuegos > 0 {
			porcentajeVictorias = float64(victorias) / float64(totalJuegos) * 100
		}

		// Determinar tipo de cliente
		tipoCliente := "nuevo"
		if totalJuegos > 10 {
			tipoCliente = "frecuente"
		} else if totalJuegos > 3 {
			tipoCliente = "ocasional"
		}

		// Contar vouchers por estado
		vouchersGenerados := len(cliente.Vouchers)
		vouchersUsados := 0
		vouchersPendientes := 0
		var ultimoVoucher *models.Voucher

		for _, voucher := range cliente.Vouchers {
			if voucher.Usado {
				vouchersUsados++
			} else {
				vouchersPendientes++
			}
			if ultimoVoucher == nil || voucher.FechaEmision.After(ultimoVoucher.FechaEmision) {
				ultimoVoucher = &voucher
			}
		}

		result = append(result, &models.ClienteConEstadisticas{
			Cliente:                     cliente,
			VouchersGenerados:           vouchersGenerados,
			VouchersUsados:              vouchersUsados,
			VouchersPendientes:          vouchersPendientes,
			PorcentajeVictoriasPersonal: porcentajeVictorias,
			TipoCliente:                 tipoCliente,
			UltimoVoucher:               ultimoVoucher,
		})
	}

	return result, nil
}

// ContarClientesPorTipo cuenta clientes por tipo
func (r *ClienteRepository) ContarClientesPorTipo(tipo string) (int, error) {
	var count int64
	query := r.db.Model(&models.Cliente{})

	switch tipo {
	case "nuevo":
		query = query.Where("total_juegos <= 3")
	case "ocasional":
		query = query.Where("total_juegos > 3 AND total_juegos <= 10")
	case "frecuente":
		query = query.Where("total_juegos > 10")
	default:
		return 0, fmt.Errorf("tipo de cliente no válido: %s", tipo)
	}

	err := query.Count(&count).Error
	return int(count), err
}

// ListarTodos lista todos los clientes
func (r *ClienteRepository) ListarTodos() ([]*models.Cliente, error) {
	var clientes []*models.Cliente
	err := r.db.Find(&clientes).Error
	return clientes, err
}

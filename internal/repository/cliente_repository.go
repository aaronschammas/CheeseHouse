package repository

import (
	"CheeseHouse/internal/models"

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

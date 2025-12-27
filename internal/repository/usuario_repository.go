package repository

import (
	"fmt"

	"gorm.io/gorm"

	"CheeseHouse/internal/models"
)

// UsuarioRepository define la interfaz para operaciones con usuarios
type UsuarioRepository interface {
	// CRUD básico
	Crear(usuario *models.Usuario) error
	BuscarPorID(id uint) (*models.Usuario, error)
	BuscarPorEmail(email string) (*models.Usuario, error)
	Actualizar(usuario *models.Usuario) error
	Eliminar(id uint) error
	ListarTodos() ([]*models.Usuario, error)

	// Consultas específicas de usuarios
	ListarPorRol(rolID uint) ([]*models.Usuario, error)
	ListarActivos() ([]*models.Usuario, error)
	BuscarPorNombre(nombre string) ([]*models.Usuario, error)

	// Roles
	BuscarRolPorID(id uint) (*models.Rol, error)
	BuscarRolPorNombre(nombre string) (*models.Rol, error)
	ListarRoles() ([]*models.Rol, error)
	CrearRol(rol *models.Rol) error

	// Contadores y estadísticas
	ContarUsuarios() (int, error)
	ContarUsuariosActivos() (int, error)
	ContarUsuariosPorRol(rolID uint) (int, error)
}

// usuarioRepository implementación de UsuarioRepository
type usuarioRepository struct {
	db *gorm.DB
}

// NewUsuarioRepository crea una nueva instancia del repositorio de usuarios
func NewUsuarioRepository(db *gorm.DB) UsuarioRepository {
	return &usuarioRepository{db: db}
}

// Crear crea un nuevo usuario en la base de datos
func (r *usuarioRepository) Crear(usuario *models.Usuario) error {
	if err := r.db.Create(usuario).Error; err != nil {
		return fmt.Errorf("error creando usuario: %w", err)
	}
	return nil
}

// BuscarPorID busca un usuario por su ID
func (r *usuarioRepository) BuscarPorID(id uint) (*models.Usuario, error) {
	var usuario models.Usuario
	if err := r.db.Preload("Rol").First(&usuario, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con ID %d no encontrado", id)
		}
		return nil, fmt.Errorf("error buscando usuario: %w", err)
	}
	return &usuario, nil
}

// BuscarPorEmail busca un usuario por su email
func (r *usuarioRepository) BuscarPorEmail(email string) (*models.Usuario, error) {
	var usuario models.Usuario
	if err := r.db.Preload("Rol").Where("email = ?", email).First(&usuario).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuario con email %s no encontrado", email)
		}
		return nil, fmt.Errorf("error buscando usuario por email: %w", err)
	}
	return &usuario, nil
}

// Actualizar actualiza los datos de un usuario
func (r *usuarioRepository) Actualizar(usuario *models.Usuario) error {
	if err := r.db.Save(usuario).Error; err != nil {
		return fmt.Errorf("error actualizando usuario: %w", err)
	}
	return nil
}

// Eliminar elimina un usuario (soft delete)
func (r *usuarioRepository) Eliminar(id uint) error {
	if err := r.db.Delete(&models.Usuario{}, id).Error; err != nil {
		return fmt.Errorf("error eliminando usuario: %w", err)
	}
	return nil
}

// ListarTodos obtiene todos los usuarios
func (r *usuarioRepository) ListarTodos() ([]*models.Usuario, error) {
	var usuarios []*models.Usuario
	if err := r.db.Preload("Rol").Find(&usuarios).Error; err != nil {
		return nil, fmt.Errorf("error listando usuarios: %w", err)
	}
	return usuarios, nil
}

// ListarPorRol obtiene usuarios de un rol específico
func (r *usuarioRepository) ListarPorRol(rolID uint) ([]*models.Usuario, error) {
	var usuarios []*models.Usuario
	if err := r.db.Preload("Rol").Where("rol_id = ?", rolID).Find(&usuarios).Error; err != nil {
		return nil, fmt.Errorf("error listando usuarios por rol: %w", err)
	}
	return usuarios, nil
}

// ListarActivos obtiene todos los usuarios activos
func (r *usuarioRepository) ListarActivos() ([]*models.Usuario, error) {
	var usuarios []*models.Usuario
	if err := r.db.Preload("Rol").Where("activo = TRUE").Find(&usuarios).Error; err != nil {
		return nil, fmt.Errorf("error listando usuarios activos: %w", err)
	}
	return usuarios, nil
}

// BuscarPorNombre busca usuarios por nombre (búsqueda parcial)
func (r *usuarioRepository) BuscarPorNombre(nombre string) ([]*models.Usuario, error) {
	var usuarios []*models.Usuario
	if err := r.db.Preload("Rol").Where("nombre LIKE ?", "%"+nombre+"%").Find(&usuarios).Error; err != nil {
		return nil, fmt.Errorf("error buscando usuarios por nombre: %w", err)
	}
	return usuarios, nil
}

// BuscarRolPorID busca un rol por su ID
func (r *usuarioRepository) BuscarRolPorID(id uint) (*models.Rol, error) {
	var rol models.Rol
	if err := r.db.First(&rol, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("rol con ID %d no encontrado", id)
		}
		return nil, fmt.Errorf("error buscando rol: %w", err)
	}
	return &rol, nil
}

// BuscarRolPorNombre busca un rol por su nombre
func (r *usuarioRepository) BuscarRolPorNombre(nombre string) (*models.Rol, error) {
	var rol models.Rol
	if err := r.db.Where("nombre = ?", nombre).First(&rol).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("rol con nombre %s no encontrado", nombre)
		}
		return nil, fmt.Errorf("error buscando rol por nombre: %w", err)
	}
	return &rol, nil
}

// ListarRoles obtiene todos los roles disponibles
func (r *usuarioRepository) ListarRoles() ([]*models.Rol, error) {
	var roles []*models.Rol
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("error listando roles: %w", err)
	}
	return roles, nil
}

// CrearRol crea un nuevo rol
func (r *usuarioRepository) CrearRol(rol *models.Rol) error {
	if err := r.db.Create(rol).Error; err != nil {
		return fmt.Errorf("error creando rol: %w", err)
	}
	return nil
}

// ContarUsuarios cuenta el total de usuarios
func (r *usuarioRepository) ContarUsuarios() (int, error) {
	var count int64
	if err := r.db.Model(&models.Usuario{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando usuarios: %w", err)
	}
	return int(count), nil
}

// ContarUsuariosActivos cuenta usuarios activos
func (r *usuarioRepository) ContarUsuariosActivos() (int, error) {
	var count int64
	if err := r.db.Model(&models.Usuario{}).Where("activo = TRUE").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando usuarios activos: %w", err)
	}
	return int(count), nil
}

// ContarUsuariosPorRol cuenta usuarios de un rol específico
func (r *usuarioRepository) ContarUsuariosPorRol(rolID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.Usuario{}).Where("rol_id = ?", rolID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("error contando usuarios por rol: %w", err)
	}
	return int(count), nil
}

// VerificarEmailUnico verifica si un email está disponible
func (r *usuarioRepository) VerificarEmailUnico(email string, excluirID ...uint) (bool, error) {
	query := r.db.Model(&models.Usuario{}).Where("email = ?", email)

	// Si se proporciona un ID, excluirlo de la búsqueda (para actualizaciones)
	if len(excluirID) > 0 && excluirID[0] > 0 {
		query = query.Where("id != ?", excluirID[0])
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("error verificando email único: %w", err)
	}

	return count == 0, nil
}

// GetUsuariosConActividad obtiene usuarios con información de su última actividad
func (r *usuarioRepository) GetUsuariosConActividad() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			u.id,
			u.nombre,
			u.email,
			u.activo,
			u.created_at,
			r.nombre as rol_nombre,
			COUNT(v.id) as vouchers_canjeados,
			MAX(v.fecha_uso) as ultima_actividad_canje
		FROM usuarios u
		LEFT JOIN roles r ON u.rol_id = r.id
		LEFT JOIN vouchers v ON u.id = v.usuario_canje
		GROUP BY u.id, u.nombre, u.email, u.activo, u.created_at, r.nombre
		ORDER BY u.activo DESC, u.created_at DESC
	`

	var resultados []map[string]interface{}
	if err := r.db.Raw(query).Scan(&resultados).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo usuarios con actividad: %w", err)
	}

	return resultados, nil
}

// GetEstadisticasUsuarios obtiene estadísticas de usuarios por rol
func (r *usuarioRepository) GetEstadisticasUsuarios() (map[string]interface{}, error) {
	query := `
		SELECT 
			r.nombre as rol,
			COUNT(u.id) as total_usuarios,
			COUNT(CASE WHEN u.activo = TRUE THEN 1 END) as usuarios_activos,
			COUNT(CASE WHEN u.activo = FALSE THEN 1 END) as usuarios_inactivos
		FROM roles r
		LEFT JOIN usuarios u ON r.id = u.rol_id
		GROUP BY r.id, r.nombre
		ORDER BY r.nombre
	`

	type RolStats struct {
		Rol               string `json:"rol"`
		TotalUsuarios     int    `json:"total_usuarios"`
		UsuariosActivos   int    `json:"usuarios_activos"`
		UsuariosInactivos int    `json:"usuarios_inactivos"`
	}

	var stats []RolStats
	if err := r.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas de usuarios: %w", err)
	}

	// Transformar a mapa para respuesta más amigable
	resultado := map[string]interface{}{
		"por_rol": stats,
	}

	// Totales generales
	totalUsuarios, _ := r.ContarUsuarios()
	usuariosActivos, _ := r.ContarUsuariosActivos()

	resultado["totales"] = map[string]int{
		"total":     totalUsuarios,
		"activos":   usuariosActivos,
		"inactivos": totalUsuarios - usuariosActivos,
	}

	return resultado, nil
}

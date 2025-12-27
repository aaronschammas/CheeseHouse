package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"CheeseHouse/internal/models"
	"CheeseHouse/internal/repository"
)

// AuthService maneja la autenticación y autorización para CheeseHouse
type AuthService struct {
	usuarioRepo repository.UsuarioRepository
	jwtSecret   string
	expiration  time.Duration
}

// Claims estructura para JWT tokens
type Claims struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Nombre  string `json:"nombre"`
	RolID   uint   `json:"rol_id"`
	RolName string `json:"rol_name"`
	jwt.RegisteredClaims
}

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(usuarioRepo repository.UsuarioRepository, jwtSecret string) *AuthService {
	return &AuthService{
		usuarioRepo: usuarioRepo,
		jwtSecret:   jwtSecret,
		expiration:  24 * time.Hour, // 24 horas por defecto
	}
}

// Login autentica un usuario y retorna un token JWT
func (a *AuthService) Login(email, password string) (*models.LoginResponse, error) {
	log.Printf("🔐 Intento de login para: %s", email)

	// Buscar usuario por email
	usuario, err := a.usuarioRepo.BuscarPorEmail(email)
	if err != nil {
		log.Printf("❌ Usuario no encontrado: %s", email)
		return &models.LoginResponse{
			Success: false,
			Message: "Credenciales inválidas",
		}, nil
	}

	// Verificar que el usuario esté activo
	if !usuario.Activo {
		log.Printf("❌ Usuario inactivo: %s", email)
		return &models.LoginResponse{
			Success: false,
			Message: "Cuenta desactivada. Contacta al administrador.",
		}, nil
	}

	// Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(usuario.PasswordHash), []byte(password)); err != nil {
		log.Printf("❌ Contraseña incorrecta para: %s", email)
		return &models.LoginResponse{
			Success: false,
			Message: "Credenciales inválidas",
		}, nil
	}

	// Cargar información del rol
	if usuario.Rol == nil {
		rol, err := a.usuarioRepo.BuscarRolPorID(usuario.RolID)
		if err != nil {
			log.Printf("⚠️  Error cargando rol para usuario %s: %v", email, err)
		} else {
			usuario.Rol = rol
		}
	}

	// Generar token JWT
	token, err := a.GenerateToken(usuario)
	if err != nil {
		log.Printf("❌ Error generando token para %s: %v", email, err)
		return &models.LoginResponse{
			Success: false,
			Message: "Error interno del servidor",
		}, nil
	}

	log.Printf("✅ Login exitoso para: %s (%s)", email, usuario.Nombre)

	return &models.LoginResponse{
		Success: true,
		Message: fmt.Sprintf("Bienvenido %s", usuario.Nombre),
		Token:   token,
		Usuario: usuario,
	}, nil
}

// GenerateToken genera un token JWT para un usuario
func (a *AuthService) GenerateToken(usuario *models.Usuario) (string, error) {
	now := time.Now()
	expirationTime := now.Add(a.expiration)

	rolName := ""
	if usuario.Rol != nil {
		rolName = usuario.Rol.Nombre
	}

	claims := &Claims{
		UserID:  usuario.ID,
		Email:   usuario.Email,
		Nombre:  usuario.Nombre,
		RolID:   usuario.RolID,
		RolName: rolName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cheesehouse-timing",
			Subject:   fmt.Sprintf("user_%d", usuario.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("error firmando token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken valida un token JWT y retorna las claims
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("error validando token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token inválido")
}

// GetUsuarioFromToken obtiene información completa del usuario desde un token
func (a *AuthService) GetUsuarioFromToken(tokenString string) (*models.Usuario, error) {
	claims, err := a.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	usuario, err := a.usuarioRepo.BuscarPorID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("usuario no encontrado: %w", err)
	}

	if !usuario.Activo {
		return nil, errors.New("usuario desactivado")
	}

	return usuario, nil
}

// HashPassword hashea una contraseña usando bcrypt
func (a *AuthService) HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("contraseña debe tener al menos 6 caracteres")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error hasheando contraseña: %w", err)
	}

	return string(hashedBytes), nil
}

// CrearUsuario crea un nuevo usuario (solo administradores)
func (a *AuthService) CrearUsuario(nombre, email, password string, rolID uint, createdBy uint) (*models.Usuario, error) {
	// Verificar que quien crea tenga permisos
	creador, err := a.usuarioRepo.BuscarPorID(createdBy)
	if err != nil {
		return nil, fmt.Errorf("creador no encontrado: %w", err)
	}

	if !a.TienePermiso(creador, "can_manage_users") {
		return nil, errors.New("sin permisos para crear usuarios")
	}

	// Verificar que el email no esté en uso
	if _, err := a.usuarioRepo.BuscarPorEmail(email); err == nil {
		return nil, errors.New("email ya está en uso")
	}

	// Hashear contraseña
	hashedPassword, err := a.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Crear usuario
	usuario := &models.Usuario{
		Nombre:       nombre,
		Email:        email,
		PasswordHash: hashedPassword,
		RolID:        rolID,
		Activo:       true,
	}

	if err := a.usuarioRepo.Crear(usuario); err != nil {
		return nil, fmt.Errorf("error creando usuario: %w", err)
	}

	log.Printf("✅ Usuario creado: %s (%s) por %s", usuario.Email, usuario.Nombre, creador.Email)

	return usuario, nil
}

// CambiarPassword cambia la contraseña de un usuario
func (a *AuthService) CambiarPassword(userID uint, currentPassword, newPassword string) error {
	usuario, err := a.usuarioRepo.BuscarPorID(userID)
	if err != nil {
		return fmt.Errorf("usuario no encontrado: %w", err)
	}

	// Verificar contraseña actual
	if err := bcrypt.CompareHashAndPassword([]byte(usuario.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("contraseña actual incorrecta")
	}

	// Hashear nueva contraseña
	newHashedPassword, err := a.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Actualizar contraseña
	usuario.PasswordHash = newHashedPassword
	if err := a.usuarioRepo.Actualizar(usuario); err != nil {
		return fmt.Errorf("error actualizando contraseña: %w", err)
	}

	log.Printf("🔐 Contraseña cambiada para: %s", usuario.Email)

	return nil
}

// TienePermiso verifica si un usuario tiene un permiso específico
func (a *AuthService) TienePermiso(usuario *models.Usuario, permiso string) bool {
	if usuario.Rol == nil {
		// Cargar rol si no está cargado
		rol, err := a.usuarioRepo.BuscarRolPorID(usuario.RolID)
		if err != nil {
			log.Printf("⚠️  Error cargando rol: %v", err)
			return false
		}
		usuario.Rol = rol
	}

	// Admin tiene todos los permisos
	if usuario.Rol.Nombre == "admin" {
		return true
	}

	// Verificar permiso específico en el JSON de permisos
	// Por simplicidad, asumir que los permisos son un objeto JSON
	// En implementación real, parsear el JSON y verificar
	return false // Implementar parsing de JSON de permisos
}

// EsAdmin verifica si un usuario es administrador
func (a *AuthService) EsAdmin(usuario *models.Usuario) bool {
	if usuario.Rol == nil {
		rol, err := a.usuarioRepo.BuscarRolPorID(usuario.RolID)
		if err != nil {
			return false
		}
		usuario.Rol = rol
	}
	return usuario.Rol.Nombre == "admin"
}

// ListarUsuarios lista todos los usuarios (solo para admins)
func (a *AuthService) ListarUsuarios(requestedBy uint) ([]*models.Usuario, error) {
	solicitante, err := a.usuarioRepo.BuscarPorID(requestedBy)
	if err != nil {
		return nil, fmt.Errorf("solicitante no encontrado: %w", err)
	}

	if !a.EsAdmin(solicitante) {
		return nil, errors.New("sin permisos para listar usuarios")
	}

	return a.usuarioRepo.ListarTodos()
}

// ActivarDesactivarUsuario activa o desactiva un usuario
func (a *AuthService) ActivarDesactivarUsuario(userID uint, activar bool, requestedBy uint) error {
	// Verificar permisos
	solicitante, err := a.usuarioRepo.BuscarPorID(requestedBy)
	if err != nil {
		return fmt.Errorf("solicitante no encontrado: %w", err)
	}

	if !a.EsAdmin(solicitante) {
		return errors.New("sin permisos para modificar usuarios")
	}

	// No permitir desactivar el propio usuario
	if userID == requestedBy {
		return errors.New("no puedes desactivar tu propia cuenta")
	}

	// Buscar usuario a modificar
	usuario, err := a.usuarioRepo.BuscarPorID(userID)
	if err != nil {
		return fmt.Errorf("usuario no encontrado: %w", err)
	}

	// Actualizar estado
	usuario.Activo = activar
	if err := a.usuarioRepo.Actualizar(usuario); err != nil {
		return fmt.Errorf("error actualizando usuario: %w", err)
	}

	accion := "activado"
	if !activar {
		accion = "desactivado"
	}

	log.Printf("👤 Usuario %s %s por %s", usuario.Email, accion, solicitante.Email)

	return nil
}

// RefreshToken genera un nuevo token para un usuario autenticado
func (a *AuthService) RefreshToken(oldTokenString string) (string, error) {
	claims, err := a.ValidateToken(oldTokenString)
	if err != nil {
		return "", fmt.Errorf("token inválido para refresh: %w", err)
	}

	usuario, err := a.usuarioRepo.BuscarPorID(claims.UserID)
	if err != nil {
		return "", fmt.Errorf("usuario no encontrado: %w", err)
	}

	if !usuario.Activo {
		return "", errors.New("usuario desactivado")
	}

	return a.GenerateToken(usuario)
}

// GetEstadisticasAuth obtiene estadísticas de autenticación
func (a *AuthService) GetEstadisticasAuth() (map[string]interface{}, error) {
	totalUsuarios, err := a.usuarioRepo.ContarUsuarios()
	if err != nil {
		return nil, err
	}

	usuariosActivos, err := a.usuarioRepo.ContarUsuariosActivos()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_usuarios":     totalUsuarios,
		"usuarios_activos":   usuariosActivos,
		"usuarios_inactivos": totalUsuarios - usuariosActivos,
	}, nil
}

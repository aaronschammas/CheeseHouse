package models

import (
	"time"
)

// Rol define los roles de usuario en CheeseHouse
type Rol struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Nombre    string    `gorm:"unique;size:50;not null" json:"nombre"` // 'admin', 'empleado'
	Permisos  string    `gorm:"type:json" json:"permisos"`             // JSON con permisos
	CreatedAt time.Time `json:"created_at"`
}

// Usuario representa empleados y administradores de CheeseHouse
type Usuario struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Nombre       string    `gorm:"size:100;not null" json:"nombre"`
	Email        string    `gorm:"unique;size:255;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"` // No incluir en JSON
	RolID        uint      `gorm:"not null" json:"rol_id"`
	Activo       bool      `gorm:"default:true" json:"activo"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relaciones
	Rol *Rol `gorm:"foreignKey:RolID" json:"rol,omitempty"`
}

// Cliente representa clientes que juegan en CheeseHouse
type Cliente struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Nombre           string     `gorm:"size:100;not null" json:"nombre"`
	Apellido         string     `gorm:"size:100;not null" json:"apellido"`
	Telefono         string     `gorm:"unique;size:20;not null" json:"telefono"` // +5491112345678
	FechaRegistro    time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"fecha_registro"`
	FechaUltimoJuego *time.Time `json:"fecha_ultimo_juego,omitempty"` // NULL si nunca jugó
	TotalJuegos      int        `gorm:"default:0" json:"total_juegos"`
	JuegosGanados    int        `gorm:"default:0" json:"juegos_ganados"`
	JuegosPerdidos   int        `gorm:"default:0" json:"juegos_perdidos"`
	Estado           string     `gorm:"type:enum('activo','bloqueado');default:'activo'" json:"estado"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relaciones
	Vouchers []Voucher `gorm:"foreignKey:ClienteID" json:"vouchers,omitempty"`
}

// Voucher representa cupones de descuento de CheeseHouse
type Voucher struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Codigo           string     `gorm:"unique;size:20;not null" json:"codigo"` // CH12345678
	ClienteID        uint       `gorm:"not null" json:"cliente_id"`
	Tipo             string     `gorm:"type:enum('juego_ganado','juego_perdido','cliente_promocion');not null" json:"tipo"`
	Descuento        int        `gorm:"not null" json:"descuento"` // Porcentaje 1-100
	Ganado           *bool      `json:"ganado,omitempty"`          // NULL para promociones, true/false para juegos
	FechaEmision     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"fecha_emision"`
	FechaVencimiento time.Time  `gorm:"not null" json:"fecha_vencimiento"`
	FechaUso         *time.Time `json:"fecha_uso,omitempty"`
	Usado            bool       `gorm:"default:false" json:"usado"`
	UsuarioCanje     *uint      `json:"usuario_canje,omitempty"` // ID del empleado que procesó el canje
	Notas            string     `gorm:"type:text" json:"notas,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`

	// Relaciones
	Cliente         *Cliente `gorm:"foreignKey:ClienteID" json:"cliente,omitempty"`
	UsuarioQueCanje *Usuario `gorm:"foreignKey:UsuarioCanje" json:"usuario_que_canje,omitempty"`
}

// CampanaClientesVouchers representa campañas promocionales
type CampanaClientesVouchers struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Nombre           string    `gorm:"size:200;not null" json:"nombre"`
	Descripcion      string    `gorm:"type:text" json:"descripcion,omitempty"`
	Descuento        int       `gorm:"not null" json:"descuento"` // Porcentaje 1-100
	FechaVencimiento time.Time `gorm:"not null" json:"fecha_vencimiento"`
	Mensaje          string    `gorm:"type:text" json:"mensaje,omitempty"`
	CreatedBy        uint      `gorm:"not null" json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Activa           bool      `gorm:"default:true" json:"activa"`

	// Relaciones
	CreadoPor *Usuario                 `gorm:"foreignKey:CreatedBy" json:"creado_por,omitempty"`
	Envios    []ClientesVouchersEnvios `gorm:"foreignKey:CampanaID" json:"envios,omitempty"`
}

// ClientesVouchersEnvios representa envíos de campañas promocionales
type ClientesVouchersEnvios struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CampanaID     uint      `gorm:"not null" json:"campana_id"`
	ClienteID     uint      `gorm:"not null" json:"cliente_id"`
	VoucherID     *uint     `json:"voucher_id,omitempty"` // NULL hasta que se genere el voucher
	CodigoVoucher string    `gorm:"size:20" json:"codigo_voucher,omitempty"`
	EnviadoAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"enviado_at"`
	Estado        string    `gorm:"type:enum('enviado','entregado','fallido');default:'enviado'" json:"estado"`
	ErrorMensaje  string    `gorm:"type:text" json:"error_mensaje,omitempty"`
	IntentosEnvio int       `gorm:"default:1" json:"intentos_envio"`

	// Relaciones
	Campana *CampanaClientesVouchers `gorm:"foreignKey:CampanaID" json:"campana,omitempty"`
	Cliente *Cliente                 `gorm:"foreignKey:ClienteID" json:"cliente,omitempty"`
	Voucher *Voucher                 `gorm:"foreignKey:VoucherID" json:"voucher,omitempty"`
}

// Pedido representa pedidos recibidos por WhatsApp (futuro)
type Pedido struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ClienteID   uint      `gorm:"not null" json:"cliente_id"`
	Telefono    string    `gorm:"size:20;not null" json:"telefono"` // Por si el cliente no está registrado
	Mensaje     string    `gorm:"type:text;not null" json:"mensaje"`
	Estado      string    `gorm:"type:enum('pendiente','procesando','completado','cancelado');default:'pendiente'" json:"estado"`
	Total       *float64  `json:"total,omitempty"`        // Monto del pedido si se calcula
	Notas       string    `gorm:"type:text" json:"notas"` // Notas del empleado
	AtendidoPor *uint     `json:"atendido_por,omitempty"` // ID del empleado que atendió
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relaciones
	Cliente         *Cliente `gorm:"foreignKey:ClienteID" json:"cliente,omitempty"`
	EmpleadoAtiende *Usuario `gorm:"foreignKey:AtendidoPor" json:"empleado_atiende,omitempty"`
}

// GameResult representa el resultado de un juego (para DTOs)
type GameResult struct {
	ClienteData ClienteData `json:"cliente"`
	Resultado   Resultado   `json:"resultado"`
}

// ClienteData datos del cliente para el juego
type ClienteData struct {
	Nombre   string `json:"nombre" binding:"required,min=2,max=50"`
	Apellido string `json:"apellido" binding:"required,min=2,max=50"`
	Telefono string `json:"telefono" binding:"required"`
}

// Resultado datos del resultado del juego
type Resultado struct {
	Gano           bool    `json:"gano"`
	TiempoObjetivo float64 `json:"tiempo_objetivo" binding:"required,min=5,max=20"`
	TiempoObtenido float64 `json:"tiempo_obtenido" binding:"required,min=0"`
	Tolerancia     float64 `json:"tolerancia,omitempty"` // Calculado por el servidor
}

// VoucherResponse respuesta al generar un voucher
type VoucherResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	Codigo             string `json:"codigo,omitempty"`
	Descuento          int    `json:"descuento,omitempty"`
	FechaVencimiento   string `json:"fecha_vencimiento,omitempty"`
	NecesitaAprobacion bool   `json:"necesita_aprobacion,omitempty"`
	ClienteID          uint   `json:"cliente_id,omitempty"`
	EsClienteNuevo     bool   `json:"es_cliente_nuevo,omitempty"`
}

// EstadisticasGenerales estadísticas del dashboard
type EstadisticasGenerales struct {
	TotalClientes       int     `json:"total_clientes"`
	TotalPartidas       int     `json:"total_partidas"`
	TotalVictorias      int     `json:"total_victorias"`
	TotalDerrotas       int     `json:"total_derrotas"`
	PorcentajeVictorias float64 `json:"porcentaje_victorias"`
	ClientesFrecuentes  int     `json:"clientes_frecuentes"`
	JugaronHoy          int     `json:"jugaron_hoy"`
	JugaronSemana       int     `json:"jugaron_semana"`
	VouchersActivos     int     `json:"vouchers_activos"`
	VouchersVencidos    int     `json:"vouchers_vencidos"`
}

// EstadisticasPorPeriodo estadísticas diarias/mensuales
type EstadisticasPorPeriodo struct {
	Fecha               string  `json:"fecha"`
	VictoriasDia        int     `json:"victorias_dia"`
	DerrotasDia         int     `json:"derrotas_dia"`
	TotalJuegosDia      int     `json:"total_juegos_dia"`
	PorcentajeVictorias float64 `json:"porcentaje_victorias_dia"`
}

// ClienteConEstadisticas cliente con sus estadísticas completas
type ClienteConEstadisticas struct {
	Cliente
	VouchersGenerados           int      `json:"vouchers_generados"`
	VouchersUsados              int      `json:"vouchers_usados"`
	VouchersPendientes          int      `json:"vouchers_pendientes"`
	PorcentajeVictoriasPersonal float64  `json:"porcentaje_victorias_personal"`
	TipoCliente                 string   `json:"tipo_cliente"` // 'nuevo', 'ocasional', 'frecuente'
	UltimoVoucher               *Voucher `json:"ultimo_voucher,omitempty"`
}

// WhatsAppMessage estructura para enviar mensajes por WhatsApp
type WhatsAppMessage struct {
	MessagingProduct string    `json:"messaging_product"`
	To               string    `json:"to"`
	Type             string    `json:"type"`
	Text             *TextBody `json:"text,omitempty"`
	Template         *Template `json:"template,omitempty"`
}

// TextBody para mensajes de texto simple
type TextBody struct {
	Body string `json:"body"`
}

// Template para mensajes con template de WhatsApp
type Template struct {
	Name       string      `json:"name"`
	Language   Language    `json:"language"`
	Components []Component `json:"components,omitempty"`
}

// Language idioma del template
type Language struct {
	Code string `json:"code"` // 'es' para español
}

// Component componentes del template
type Component struct {
	Type       string      `json:"type"` // 'body', 'header', 'button'
	Parameters []Parameter `json:"parameters,omitempty"`
}

// Parameter parámetros del template
type Parameter struct {
	Type string `json:"type"` // 'text'
	Text string `json:"text"`
}

// WhatsAppWebhookMessage mensaje recibido por webhook
type WhatsAppWebhookMessage struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// LoginRequest request para login de empleados
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse respuesta del login
type LoginResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Token   string   `json:"token,omitempty"`
	Usuario *Usuario `json:"usuario,omitempty"`
}

// CanjearVoucherRequest request para canjear voucher
type CanjearVoucherRequest struct {
	Codigo string `json:"codigo" binding:"required,min=6,max=20"`
}

// CanjearVoucherResponse respuesta del canje
type CanjearVoucherResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Descuento int    `json:"descuento,omitempty"`
	Cliente   string `json:"cliente,omitempty"`
}

// TableName especifica nombres de tabla personalizados para GORM
func (Rol) TableName() string                     { return "roles" }
func (Usuario) TableName() string                 { return "usuarios" }
func (Cliente) TableName() string                 { return "clientes" }
func (Voucher) TableName() string                 { return "vouchers" }
func (CampanaClientesVouchers) TableName() string { return "campañas_clientes_vouchers" }
func (ClientesVouchersEnvios) TableName() string  { return "clientes_vouchers_envios" }
func (Pedido) TableName() string                  { return "pedidos" }

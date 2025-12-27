package repository

type Repositories struct {
	Cliente ClienteRepository
	Voucher VoucherRepository
	Usuario UsuarioRepository
	Campana CampanaRepository
}

// NewRepositories crea una nueva instancia con todos los repositorios
func NewRepositories(
	cliente ClienteRepository,
	voucher VoucherRepository,
	usuario UsuarioRepository,
	campana CampanaRepository,
) *Repositories {
	return &Repositories{
		Cliente: cliente,
		Voucher: voucher,
		Usuario: usuario,
		Campana: campana,
	}
}
